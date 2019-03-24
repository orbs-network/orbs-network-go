// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"math/big"
	"strings"
	"testing"
)

//TODO (v1) move this file back to ethereum/adapter package

func TestEthereumNodeAdapter_CallContract(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		adapter, auth, commit := createSimulator(t)
		t.Run("Simulator Adapter", testCallContract(ctx, adapter, auth, commit))

		if runningWithDocker() {
			adapter, auth, commit = createRpcClient(t)
			t.Run("RPC Adapter", testCallContract(ctx, adapter, auth, commit))
		} else {
			t.Skip("skipping, external tests disabled")
		}
	})
}

func TestEthereumNodeAdapter_GetLogs(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		adapter, auth, commit := createSimulator(t)
		t.Run("Simulator Adapter", testGetLogs(ctx, adapter, auth, commit))

		if runningWithDocker() {
			adapter, auth, commit = createRpcClient(t)
			t.Run("RPC Adapter", testGetLogs(ctx, adapter, auth, commit))
		} else {
			t.Skip("skipping, external tests disabled")
		}
	})
}

func testGetLogs(ctx context.Context, adapter adapter.DeployingEthereumConnection, auth *bind.TransactOpts, commit func()) func(t *testing.T) {
	return func(t *testing.T) {
		parsedABI, err := abi.JSON(strings.NewReader(contract.EmitEventAbi))
		require.NoError(t, err, "failed parsing ABI")

		contractAddress, emitEventContract, err := adapter.DeployEthereumContract(auth, contract.EmitEventAbi, contract.EmitEventBin)
		commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		tuid := big.NewInt(17)
		ethAddress := common.HexToAddress("80755fE3D774006c9A9563A09310a0909c42C786")
		orbsAddress := [20]byte{0x1, 0x2, 0x3}
		amount := big.NewInt(42)

		tx, err := emitEventContract.Transact(auth, "transferOut", tuid, ethAddress, orbsAddress, amount)
		commit()
		require.NoError(t, err, "failed emitting event")

		eventABI := parsedABI.Events["TransferredOut"]
		eventSignature := eventABI.Id().Bytes()

		logs, err := adapter.GetTransactionLogs(ctx, primitives.Uint256(tx.Hash().Bytes()), eventSignature)
		require.NoError(t, err, "failed getting logs")

		require.Len(t, logs, 1, "did not get the expected event log")
		log := logs[0]

		require.Equal(t, contractAddress.Bytes(), log.ContractAddress, "contract address in log differed from actual contract address")
		require.Equal(t, eventSignature, log.PackedTopics[0], "event returned did not have the expected signature as the first topic")

		data, err := eventABI.Inputs.UnpackValues(log.Data)
		require.NoError(t, err, "failed unpacking data")

		require.Len(t, data, 1, "got unexpected items in log data")
		require.EqualValues(t, amount, data[0], "did not get expected value from event")

		outTuid := big.NewInt(0)
		outTuid.SetBytes(log.PackedTopics[1])
		require.EqualValues(t, tuid, outTuid, "failed unpacking tuid")

		eventEthAddress := log.PackedTopics[2][32-len(ethAddress):]
		require.EqualValues(t, ethAddress.Bytes(), eventEthAddress, "failed unpacking ethAddress")
	}
}

func testCallContract(ctx context.Context, adapter adapter.DeployingEthereumConnection, auth *bind.TransactOpts, commit func()) func(t *testing.T) {
	return func(t *testing.T) {
		address, err := adapter.DeploySimpleStorageContract(auth, "foobar")
		commit()
		require.NoError(t, err, "failed deploying contract to Ethereum")

		parsedABI, err := abi.JSON(strings.NewReader(contract.SimpleStorageABI))
		require.NoError(t, err, "failed parsing ABI")

		packedInput, err := parsedABI.Pack("getString")
		require.NoError(t, err, "failed packing arguments")

		packedOutput, err := adapter.CallContract(ctx, address, packedInput, nil)

		var out string
		err = parsedABI.Unpack(&out, "getString", packedOutput)
		require.NoError(t, err, "could not unpack call output")

		require.Equal(t, "foobar", out, "string output differed from expected")
	}
}

func createRpcClient(tb testing.TB) (adapter.DeployingEthereumConnection, *bind.TransactOpts, func()) {
	cfg := ConfigForExternalRPCConnection()

	a := adapter.NewEthereumRpcConnection(cfg, log.DefaultTestingLogger(tb))
	auth, err := cfg.GetAuthFromConfig()
	if err != nil {
		panic(err)
	}

	return a, auth, func() {}
}

func createSimulator(tb testing.TB) (adapter.DeployingEthereumConnection, *bind.TransactOpts, func()) {
	a := adapter.NewEthereumSimulatorConnection(log.DefaultTestingLogger(tb))
	opts := a.GetAuth()
	commit := a.Commit

	return a, opts, commit
}
