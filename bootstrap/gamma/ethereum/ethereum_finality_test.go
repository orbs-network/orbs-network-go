// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ethereum

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/bootstrap/gamma"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/e2e/contracts/calc/eth"
	"github.com/stretchr/testify/require"
	"os"
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

		gammaEndpoint := gamma.RunOnRandomPort(t, fmt.Sprintf(`{"ethereum_endpoint":"%s"}`, ethereumEndpoint))

		contractOwner, _ := orbsClient.CreateAccount()
		orbs := orbsClient.NewClient(gammaEndpoint, 42, codec.NETWORK_TYPE_TEST_NET)

		key, err := crypto.HexToECDSA(privateKey)
		require.NoError(t, err, "failed reading Ethereum private key")

		auth := bind.NewKeyedTransactor(key)

		ethRpc, err := rpc.DialContext(ctx, ethereumEndpoint)
		require.NoError(t, err, "failed connecting to Ganache")
		ethereumRpc := ethclient.NewClient(ethRpc)

		ensureFinalityInGanacheAndGamma(t, gammaEndpoint, ethRpc) // move to the future a bit to ensure finality before the test even starts interacting with Ganache

		loggerContractAddress, _, loggerContract, err := eth.DeployLogger(auth, ethereumRpc)
		require.NoError(t, err, "failed deploying Logger contract to Ganache")

		gamma.DeployContract(t, orbs, contractOwner, "LogCalculator", []byte(ORBS_CALC_CONTRACT))

		txHashes := sendEthTransactions(t, loggerContract, auth, 25)

		ensureFinalityInGanacheAndGamma(t, gammaEndpoint, ethRpc)

		res := gamma.SendTransaction(t, orbs, contractOwner, "LogCalculator", "bind", loggerContractAddress.Bytes(), []byte(eth.LoggerABI)) // this happens AFTER moving time forwards so that a block with the new time is closed

		queryRes := gamma.SendQuery(t, orbs, contractOwner, res.BlockHeight, "LogCalculator", "sum", strings.Join(txHashes, ","))

		require.EqualValues(t, 325, queryRes.OutputArguments[0], "did not get expected logs from Ethereum")
	})

}

func ensureFinalityInGanacheAndGamma(t *testing.T, gammaEndpoint string, ethRpc *rpc.Client) {
	finalityTime := 120 * time.Second
	gamma.TimeTravel(t, gammaEndpoint, finalityTime)
	moveBlocksInGanache(t, ethRpc, int(finalityTime.Seconds()), 1)
}

func sendEthTransactions(t testing.TB, loggerContract *eth.Logger, auth *bind.TransactOpts, numOfTxs int) (hashes []string) {
	for i := 1; i <= numOfTxs; i++ {
		tx, err := loggerContract.Log(auth, int32(i))
		require.NoError(t, err, "failed sending Ethereum tx")
		hashes = append(hashes, tx.Hash().String())
	}

	return
}
