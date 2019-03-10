package e2e

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/e2e/contracts/calc/eth"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const ORBS_CALC_CONTRACT = `
package main

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

var PUBLIC = sdk.Export(sum, bind)
var SYSTEM = sdk.Export(_init)

func _init() {
}

var ethAddressKey = []byte("ETH_CONTRACT_ADDRESS")
var ethABIKey = []byte("ETH_CONTRACT_ABI")

func bind(ethContractAddress []byte, abi []byte) {
	state.WriteString(ethAddressKey, "0x"+hex.EncodeToString(ethContractAddress))
	state.WriteString(ethABIKey, string(abi))
}

func sum(tx1 string, tx2 string, tx3 string) uint64 {
	abi := state.ReadString(ethABIKey)
	address := state.ReadString(ethAddressKey)
	if abi == "" || address == "" {
		panic("Trying to read from an unbound contract")
	}

	var sum uint64
	for _, txHash := range []string{tx1, tx2, tx3} {
		var out struct {
			Count int32
		}

		ethereum.GetTransactionLog(address, abi, txHash, "Log", &out)
		sum += uint64(out.Count)

	}

	return sum
}`

func TestReadFromEthereumLogsTakingFinalityIntoAccount(t *testing.T) {
	privateKey := os.Getenv("ETHEREUM_PRIVATE_KEY")
	ethereumEndpoint := os.Getenv("ETHEREUM_ENDPOINT")

	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}

	if ethereumEndpoint == "" || privateKey == "" {
		t.Skip("Skipping Ganache-dependent E2E - missing endpoint or private key")
	}

	test.WithContext(func(ctx context.Context) {

		h := newHarness()
		h.waitUntilTransactionPoolIsReady(t)

		contractOwner, _ := orbsClient.CreateAccount()

		key, err := crypto.HexToECDSA(privateKey)
		require.NoError(t, err, "failed reading Ethereum private key")

		auth := bind.NewKeyedTransactor(key)

		ethRpc, err := rpc.DialContext(ctx, ethereumEndpoint)
		require.NoError(t, err, "failed connecting to Ganache")
		ethereumRpc := ethclient.NewClient(ethRpc)

		loggerContractAddress, _, loggerContract, err := eth.DeployLogger(auth, ethereumRpc)
		require.NoError(t, err, "failed deploying Logger contract to Ganache")

		h.eventuallyDeploy(t, keys.NewEd25519KeyPair(contractOwner.PublicKey, contractOwner.PrivateKey), "LogCalculator", []byte(ORBS_CALC_CONTRACT))

		res, _, err := h.sendTransaction(contractOwner.PublicKey, contractOwner.PrivateKey, "LogCalculator", "bind", loggerContractAddress.Bytes(), []byte(eth.LoggerABI))
		require.NoError(t, err, "failed binding Ethereum contract to Orbs contract")
		require.EqualValues(t, codec.TRANSACTION_STATUS_COMMITTED.String(), res.TransactionStatus.String(), "deployment transaction not committed")
		require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), res.ExecutionResult.String(), "deployment transaction not successful")

		tx1, err := loggerContract.Log(auth, int32(1))
		require.NoError(t, err, "failed sending Ethereum tx")
		tx2, err := loggerContract.Log(auth, int32(2))
		require.NoError(t, err, "failed sending Ethereum tx")
		tx3, err := loggerContract.Log(auth, int32(3))
		require.NoError(t, err, "failed sending Ethereum tx")

		ethBlockNumber := int64(0)
		latestBlockTime := time.Unix(0, 0)
		now := time.Now()
		for ethBlockNumber < 100 || latestBlockTime.Before(now) {
			ethBlock, err := ethereumRpc.HeaderByNumber(ctx, nil)
			require.NoError(t, err, "failed getting Ethereum block number")
			require.NoError(t, ethRpc.CallContext(ctx, struct{}{}, "evm_mine"), "failed mining block")
			require.NoError(t, ethRpc.CallContext(ctx, struct{}{}, "evm_increaseTime", 10), "failed increasing time")
			ethBlockNumber = ethBlock.Number.Int64()
			latestBlockTime = time.Unix(ethBlock.Time.Int64(), 0)
		}

		queryRes, err := h.runQuery(contractOwner.PublicKey, "LogCalculator", "sum", tx1.Hash().String(), tx2.Hash().String(), tx3.Hash().String())
		require.NoError(t, err, "failed reading log")
		require.EqualValues(t, codec.REQUEST_STATUS_COMPLETED.String(), queryRes.RequestStatus.String(), "failed calling sum method")
		require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), queryRes.ExecutionResult.String(), "failed calling sum method")

		require.EqualValues(t, 6, queryRes.OutputArguments[0], "did not get expected logs from Ethereum")
	})

}

func readFile(path string) ([]byte, error) {
	absPath, _ := filepath.Abs(path)
	return ioutil.ReadFile(absPath)
}
