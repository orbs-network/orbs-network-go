package e2e

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"testing"
)

func TestReadFromEthereumLogsTakingFinalityIntoAccount(t *testing.T) {
	privateKey := "f2ce3a9eddde6e5d996f6fe7c1882960b0e8ee8d799e0ef608276b8de4dc7f19"
	ethereumEndpoint := "http://127.0.0.1:7545"

	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	test.WithContext(func(ctx context.Context) {

		h := newHarness()
		h.waitUntilTransactionPoolIsReady(t)

		contractOwner, _ := orbsClient.CreateAccount()

		key, err := crypto.HexToECDSA(privateKey)
		require.NoError(t, err, "failed reading Ethereum private key")

		auth := bind.NewKeyedTransactor(key)

		abiJson, err := readFile("./Contracts/EthereumLogger/logger_sol_Logger.abi")
		require.NoError(t, err, "failed reading ABI from file")
		//
		//bytecode, err := readFile("./Contracts/EthereumLogger/logger_sol_Logger.bin")
		//require.NoError(t, err, "failed reading bytecode from file")

		parsedAbi, err := abi.JSON(bytes.NewReader(abiJson))

		ethRpc, err := rpc.DialContext(ctx, ethereumEndpoint)
		require.NoError(t, err, "failed connecting to Ganache")
		ethereumRpc := ethclient.NewClient(ethRpc)

		//loggerContractAddress, _, loggerContract, err := bind.DeployContract(auth, parsedAbi, bytecode, ethereumRpc)
		//require.NoError(t, err, "failed deploying Logger contract to Ganache")

		//loggerContractAddress, err := deployEthereumContractManually(ctx, ethereumRpc, auth, parsedAbi, bytecode)
		//require.NoError(t, err, "failed deploying Logger contract to Ganache")
		loggerContractAddress := common.HexToAddress("0x5cd0D270C30EDa5ADa6b45a5289AFF1D425759b3")
		loggerContract := bind.NewBoundContract(loggerContractAddress, parsedAbi, ethereumRpc, ethereumRpc, ethereumRpc)

		orbsLogReaderCode, err := readFile("./Contracts/EthereumLogger/logger.go")
		require.NoError(t, err, "failed reading Orbs contract from file")

		h.eventuallyDeploy(t, keys.NewEd25519KeyPair(contractOwner.PublicKey, contractOwner.PrivateKey), "LogReader", orbsLogReaderCode)

		res, _, err := h.sendTransaction(contractOwner.PublicKey, contractOwner.PrivateKey, "LogReader", "bind", loggerContractAddress.Bytes(), abiJson)
		require.NoError(t, err, "failed binding Ethereum contract to Orbs contract")
		require.EqualValues(t, codec.TRANSACTION_STATUS_COMMITTED.String(), res.TransactionStatus.String(), "deployment transaction not committed")
		require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), res.ExecutionResult.String(), "deployment transaction not successful")

		tx1, err := loggerContract.Transact(auth, "log", int32(1))
		require.NoError(t, err, "failed sending Ethereum tx")
		tx2, err := loggerContract.Transact(auth, "log", int32(2))
		require.NoError(t, err, "failed sending Ethereum tx")
		tx3, err := loggerContract.Transact(auth, "log", int32(3))
		require.NoError(t, err, "failed sending Ethereum tx")

		ethBlockNumber := int64(0)
		for ethBlockNumber < 90 {
			ethBlock, err := ethereumRpc.HeaderByNumber(ctx, nil)
			require.NoError(t, err, "failed getting Ethereum block number")
			require.NoError(t, ethRpc.CallContext(ctx, struct{}{}, "evm_mine"), "failed mining block")
			ethBlockNumber = ethBlock.Number.Int64()
		}

		queryRes, err := h.runQuery(contractOwner.PublicKey, "LogReader", "read", tx1.Hash().String(), tx2.Hash().String(), tx3.Hash().String())
		require.NoError(t, err, "failed reading log")
		require.EqualValues(t, codec.REQUEST_STATUS_COMPLETED.String(), queryRes.RequestStatus.String(), "deployment transaction not committed")
		require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), queryRes.ExecutionResult.String(), "deployment transaction not successful")

		require.EqualValues(t, 6, queryRes.OutputArguments[0], "did not get expected logs from Ethereum")
	})

}

func readFile(path string) ([]byte, error) {
	absPath, _ := filepath.Abs(path)
	return ioutil.ReadFile(absPath)
}

func deployEthereumContractManually(ctx context.Context, c *ethclient.Client, auth *bind.TransactOpts, parsedAbi abi.ABI, bytecode []byte, params ...interface{}) (*common.Address, error) {

	input, err := parsedAbi.Pack("", params...)
	if err != nil {
		return nil, err
	}

	data := append(bytecode, input...)

	nonce, err := c.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve account nonce: %s", err)
	}

	rawTx := types.NewContractCreation(nonce, big.NewInt(0), 300000000, big.NewInt(1), data)
	signedTx, err := auth.Signer(types.HomesteadSigner{}, auth.From, rawTx)
	if err != nil {
		return nil, err
	}

	err = c.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, err
	}

	contractAddress, err := bind.WaitDeployed(ctx, c, signedTx)
	if err != nil {
		return nil, err
	}

	return &contractAddress, nil
}
