// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	orbsClient "github.com/orbs-network/orbs-client-sdk-go/orbs"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEthereumConnector_GetTransactionLogs_ParsesASBEvent(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection())
			h.moveBlocksInGanache(t, ctx, 100, 1) // pad Ganache nicely so that any previous test doesn't affect this one

			auth, err := h.config.GetAuthFromConfig()
			require.NoError(t, err)
			contractAddress, deployedContract, err := h.rpcAdapter.DeployEthereumContract(auth, contract.EmitEventAbi, contract.EmitEventBin)
			require.NoError(t, err, "failed deploying contract to Ethereum")

			blockAtDeploy, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
			require.NoError(t, err, "failed to get latest block in ganache")
			t.Logf("block at deploy: %s", blockAtDeploy)

			amount := big.NewInt(42)
			tuid := big.NewInt(33)
			ethAddress := common.BigToAddress(big.NewInt(42000000000))
			orbsAddress := anOrbsAddress()

			tx, err := deployedContract.Transact(auth, "transferOut", tuid, ethAddress, orbsAddress, amount)
			require.NoError(t, err, "failed emitting event")
			blockAfterEmit, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
			t.Logf("block after emit: %s", blockAfterEmit)

			t.Logf("finality is %f seconds, %d blocks", h.config.finalityTimeComponent.Seconds(), h.config.finalityBlocksComponent)
			h.moveBlocksInGanache(t, ctx, int(h.config.finalityBlocksComponent*2), 1) // finality blocks + block we will request below of because of the finder algo

			blockAfterPad, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
			require.NoError(t, err, "failed to get latest block in ganache")
			referenceTime := time.Unix(blockAfterPad.TimeInSeconds, 0)

			t.Logf("reference time: %d", referenceTime.UnixNano())

			out, err := h.connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
				EthereumContractAddress: contractAddress.Hex(),
				EthereumTxhash:          tx.Hash().Hex(),
				EthereumEventName:       "TransferredOut",
				EthereumJsonAbi:         contract.EmitEventAbi,
				ReferenceTimestamp:      primitives.TimestampNano(referenceTime.UnixNano()),
			})
			require.NoError(t, err, "failed getting logs")

			parsedABI, err := abi.JSON(strings.NewReader(contract.EmitEventAbi))
			require.NoError(t, err, "failed parsing ABI")

			event := new(contract.EmitEvent)
			err = ethereum.ABIUnpackAllEventArguments(parsedABI, event, "TransferredOut", out.EthereumAbiPackedOutputs[0])
			require.NoError(t, err, "failed getting amount from tx log")
			require.EqualValues(t, ethAddress, event.EthAddress, "failed getting ethAddress from unpacked data")
			require.EqualValues(t, tuid, event.Tuid, "failed getting tuid from unpacked data")
			require.EqualValues(t, orbsAddress, event.OrbsAddress, "failed getting orbsAddress from unpacked data")
			require.EqualValues(t, amount, event.Value, "failed getting amount from unpacked data")
			require.NotZero(t, out.EthereumBlockNumber, "expected returned block number to be non-zero")
			require.Equal(t, uint32(0), out.EthereumTxindex, "expected returned txIndex to zero")
		})
	})
}

func TestEthereumConnector_GetTransactionLogs_ParsesEventsWithAddressArray(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			cfg := ConfigForExternalRPCConnection()
			h := newRpcEthereumConnectorHarness(parent.Logger, cfg)
			h.moveBlocksInGanache(t, ctx, 100, 1) // pad Ganache nicely so that any previous test doesn't affect this one

			contractABI, err := readFile("../contract/EmitAddressArrayEvent_sol_EmitAddressArrayEvent.abi")
			require.NoError(t, err, "failed reading contract ABI")

			contractBin, err := readFile("../contract/EmitAddressArrayEvent_sol_EmitAddressArrayEvent.bin")
			require.NoError(t, err, "failed reading contract binary")

			auth, err := cfg.GetAuthFromConfig()
			require.NoError(t, err, "failed reading auth from config")

			contractAddress, deployedContract, err := h.rpcAdapter.DeployEthereumContract(auth, string(contractABI), string(contractBin))
			require.NoError(t, err, "failed deploying contract to Ethereum")
			blockAtDeploy, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
			require.NoError(t, err, "failed to get latest block in ganache")
			t.Logf("block at deploy: %s", blockAtDeploy)

			addresses := []common.Address{{0x1, 0x2, 0x3}, {0x4, 0x5, 0x6}, {0x7, 0x8}, {0x9}}

			tx, err := deployedContract.Transact(auth, "fire", addresses)
			require.NoError(t, err, "failed emitting event")

			t.Logf("finality is %f seconds, %d blocks", h.config.finalityTimeComponent.Seconds(), h.config.finalityBlocksComponent)
			h.moveBlocksInGanache(t, ctx, int(h.config.finalityBlocksComponent*2), 1) // finality blocks + block we will request below of because of the finder algo

			blockAfterPad, err := h.rpcAdapter.HeaderByNumber(ctx, nil)
			require.NoError(t, err, "failed to get latest block in ganache")
			referenceTime := time.Unix(blockAfterPad.TimeInSeconds, 0)

			t.Logf("reference time: %d", referenceTime.UnixNano())

			out, err := h.connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
				EthereumContractAddress: contractAddress.Hex(),
				EthereumTxhash:          tx.Hash().Hex(),
				EthereumEventName:       "Vote",
				EthereumJsonAbi:         string(contractABI),
				ReferenceTimestamp:      primitives.TimestampNano(referenceTime.UnixNano()),
			})
			require.NoError(t, err, "failed getting logs")

			parsedABI, err := abi.JSON(bytes.NewReader(contractABI))
			require.NoError(t, err, "failed parsing ABI")

			event := new(struct {
				Voter        common.Address
				Nodeslist    []common.Address
				Vote_counter *big.Int
			})
			err = ethereum.ABIUnpackAllEventArguments(parsedABI, event, "Vote", out.EthereumAbiPackedOutputs[0])
			require.NoError(t, err, "failed unpacking event")
			require.EqualValues(t, addresses, event.Nodeslist, "event did not include expected addresses")
		})
	})
}

func TestEthereumConnector_GetTransactionLogs_FailsOnWrongContract(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			cfg := ConfigForExternalRPCConnection()
			h := newRpcEthereumConnectorHarness(parent.Logger, cfg)

			auth, err := cfg.GetAuthFromConfig()
			require.NoError(t, err, "failed reading auth from config")

			contractAddress, deployedContract, err := h.rpcAdapter.DeployEthereumContract(auth, contract.EmitEventAbi, contract.EmitEventBin)
			require.NoError(t, err, "failed deploying contract to Ethereum")

			amount := big.NewInt(42)
			tuid := big.NewInt(33)
			ethAddress := common.BigToAddress(big.NewInt(42000000000))
			orbsAddress := anOrbsAddress()

			tx, err := deployedContract.Transact(auth, "transferOut", tuid, ethAddress, orbsAddress, amount)
			require.NoError(t, err, "failed emitting event")

			incorrectContractAddress := "0x6C94224Eb459535C752D2684F3654a0D71e32516" // taken from somewhere else
			require.NotEqual(t, incorrectContractAddress, contractAddress.Hex(), "contract should not accidentally match the one we use as incorrect")

			_, err = h.connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
				EthereumContractAddress: incorrectContractAddress,
				EthereumTxhash:          tx.Hash().Hex(),
				EthereumEventName:       "TransferredOut",
				EthereumJsonAbi:         contract.EmitEventAbi,
				ReferenceTimestamp:      primitives.TimestampNano(time.Now().UnixNano()),
			})
			require.Error(t, err, "should fail getting logs due to incorrect contract")
		})
	})
}

func readFile(path string) ([]byte, error) {
	absPath, _ := filepath.Abs(path)
	return ioutil.ReadFile(absPath)
}

func anOrbsAddress() [20]byte {
	orbsUser, _ := orbsClient.CreateAccount()
	var orbsUserAddress [20]byte
	copy(orbsUserAddress[:], orbsUser.AddressAsBytes())
	return orbsUserAddress
}
