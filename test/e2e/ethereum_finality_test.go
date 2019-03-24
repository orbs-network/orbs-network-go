// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	"strings"
	"testing"
	"time"
)

const ORBS_CALC_CONTRACT = `package main

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"strings"
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

func sum(txCommaSeparatedList string) uint64 {
	abi := state.ReadString(ethABIKey)
	address := state.ReadString(ethAddressKey)
	if abi == "" || address == "" {
		panic("Trying to read from an unbound contract")
	}

	var sum uint64
	for _, txHash := range strings.Split(txCommaSeparatedList, ",") {
		var out struct {
			Count int32
		}

		ethereum.GetTransactionLog(address, abi, txHash, "Log", &out)
		sum += uint64(out.Count)
	}

	return sum
}

`

func moveToRealtimeInGanache(t *testing.T, c *rpc.Client, ethC *ethclient.Client, ctx context.Context, buffer int) {
	latestBlockInGanache, err := ethC.HeaderByNumber(ctx, nil)
	require.NoError(t, err, "failed to get latest block in ganache")

	now := time.Now().Unix()
	gap := now - latestBlockInGanache.Time.Int64() - int64(buffer)
	require.True(t, gap >= 0, "ganache must be set up back enough to the past so finality test would pass in this flow, it was rolled too close to realtime, gap was %d", gap)
	t.Logf("moving %d blocks into the future to get to now - %d seconds", gap, buffer)
	moveBlocksInGanache(t, c, int(gap), 1)
}

func moveBlocksInGanache(t *testing.T, c *rpc.Client, count int, blockGapInSeconds int) {
	for i := 0; i < count; i++ {
		require.NoError(t, c.Call(struct{}{}, "evm_increaseTime", blockGapInSeconds), "failed evm_increaseTime")
		require.NoError(t, c.Call(struct{}{}, "evm_mine"), "failed evm_mine")
	}
}

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

		txHashes := sendEthTransactions(t, loggerContract, auth, 25)

		moveToRealtimeInGanache(t, ethRpc, ethereumRpc, ctx, 0)

		queryRes, err := h.runQuery(contractOwner.PublicKey, "LogCalculator", "sum", strings.Join(txHashes, ","))
		require.NoError(t, err, "failed reading log")
		require.EqualValues(t, codec.REQUEST_STATUS_COMPLETED.String(), queryRes.RequestStatus.String(), "failed calling sum method")
		require.EqualValues(t, codec.EXECUTION_RESULT_SUCCESS.String(), queryRes.ExecutionResult.String(), "failed calling sum method")

		require.EqualValues(t, 325, queryRes.OutputArguments[0], "did not get expected logs from Ethereum")
	})

}

func sendEthTransactions(t testing.TB, loggerContract *eth.Logger, auth *bind.TransactOpts, numOfTxs int) (hashes []string) {
	for i := 1; i <= numOfTxs; i++ {
		tx, err := loggerContract.Log(auth, int32(i))
		require.NoError(t, err, "failed sending Ethereum tx")
		hashes = append(hashes, tx.Hash().String())
	}

	return
}

func readFile(path string) ([]byte, error) {
	absPath, _ := filepath.Abs(path)
	return ioutil.ReadFile(absPath)
}
