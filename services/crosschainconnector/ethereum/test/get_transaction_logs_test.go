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
	test.WithContext(func(ctx context.Context) {
		h := newSimulatedEthereumConnectorHarness(t)

		contractAddress, deployedContract, err := h.simAdapter.DeployEthereumContract(h.simAdapter.GetAuth(), contract.EmitEventAbi, contract.EmitEventBin)
		h.simAdapter.Commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		amount := big.NewInt(42)
		tuid := big.NewInt(33)
		ethAddress := common.BigToAddress(big.NewInt(42000000000))
		orbsAddress := anOrbsAddress()

		tx, err := deployedContract.Transact(h.simAdapter.GetAuth(), "transferOut", tuid, ethAddress, orbsAddress, amount)
		h.simAdapter.Commit()
		require.NoError(t, err, "failed emitting event")

		out, err := h.connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
			EthereumContractAddress: contractAddress.Hex(),
			EthereumTxhash:          tx.Hash().Hex(),
			EthereumEventName:       "TransferredOut",
			EthereumJsonAbi:         contract.EmitEventAbi,
			ReferenceTimestamp:      primitives.TimestampNano(time.Now().UnixNano()),
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
}

func TestEthereumConnector_GetTransactionLogs_ParsesEventsWithAddressArray(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newSimulatedEthereumConnectorHarness(t)

		contractABI, err := readFile("../contract/EmitAddressArrayEvent_sol_EmitAddressArrayEvent.abi")
		require.NoError(t, err, "failed reading contract ABI")

		contractBin, err := readFile("../contract/EmitAddressArrayEvent_sol_EmitAddressArrayEvent.bin")
		require.NoError(t, err, "failed reading contract binary")

		contractAddress, deployedContract, err := h.simAdapter.DeployEthereumContract(h.simAdapter.GetAuth(), string(contractABI), string(contractBin))
		h.simAdapter.Commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		addresses := []common.Address{{0x1, 0x2, 0x3}, {0x4, 0x5, 0x6}, {0x7, 0x8}, {0x9}}

		tx, err := deployedContract.Transact(h.simAdapter.GetAuth(), "fire", addresses)
		h.simAdapter.Commit()
		require.NoError(t, err, "failed emitting event")

		out, err := h.connector.EthereumGetTransactionLogs(ctx, &services.EthereumGetTransactionLogsInput{
			EthereumContractAddress: contractAddress.Hex(),
			EthereumTxhash:          tx.Hash().Hex(),
			EthereumEventName:       "Vote",
			EthereumJsonAbi:         string(contractABI),
			ReferenceTimestamp:      primitives.TimestampNano(time.Now().UnixNano()),
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
}

func TestEthereumConnector_GetTransactionLogs_FailsOnWrongContract(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newSimulatedEthereumConnectorHarness(t)

		contractAddress, deployedContract, err := h.simAdapter.DeployEthereumContract(h.simAdapter.GetAuth(), contract.EmitEventAbi, contract.EmitEventBin)
		h.simAdapter.Commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		amount := big.NewInt(42)
		tuid := big.NewInt(33)
		ethAddress := common.BigToAddress(big.NewInt(42000000000))
		orbsAddress := anOrbsAddress()

		tx, err := deployedContract.Transact(h.simAdapter.GetAuth(), "transferOut", tuid, ethAddress, orbsAddress, amount)
		h.simAdapter.Commit()
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
