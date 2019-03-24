// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBenchmarkToken_GetBalancePostInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		targetAddress := builders.ClientAddressForEd25519SignerForTests(1)
		const balance = uint64(3)

		t.Log("Runs BenchmarkToken.getBalance")

		call := processCallInput().WithMethod("BenchmarkToken", "getBalance").WithArgs([]byte(targetAddress)).Build()
		h.expectSdkCallMadeWithStateRead(targetAddress, uint64ToBytes(balance))

		output, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.ArgumentsArray(balance), output.OutputArgumentArray, "call return args should be equal")
		h.verifySdkCallMade(t)
	})
}

func TestBenchmarkToken_TransferThenGetBalance(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		callerAddress := builders.ClientAddressForEd25519SignerForTests(0)
		targetAddress := builders.ClientAddressForEd25519SignerForTests(1)
		const amount, callerBalance, targetBalance = uint64(3), uint64(20), uint64(10)

		t.Log("Runs BenchmarkToken.transfer")

		call := processCallInput().WithMethod("BenchmarkToken", "transfer").WithArgs(amount, []byte(targetAddress)).Build()
		h.expectSdkCallMadeWithAddressGetCaller(callerAddress)
		h.expectSdkCallMadeWithStateRead(callerAddress, uint64ToBytes(callerBalance))
		h.expectSdkCallMadeWithStateWrite(callerAddress, uint64ToBytes(callerBalance-amount))
		h.expectSdkCallMadeWithStateRead(targetAddress, uint64ToBytes(targetBalance))
		h.expectSdkCallMadeWithStateWrite(targetAddress, uint64ToBytes(targetBalance+amount))

		output, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.ArgumentsArray(), output.OutputArgumentArray, "call return args should be equal")
		h.verifySdkCallMade(t)

		t.Log("Runs BenchmarkToken.getBalance")

		call = processCallInput().WithMethod("BenchmarkToken", "getBalance").WithArgs([]byte(callerAddress)).Build()
		h.expectSdkCallMadeWithStateRead(callerAddress, uint64ToBytes(callerBalance-amount))

		output, err = h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.ArgumentsArray(callerBalance-amount), output.OutputArgumentArray, "call return args should be equal")
		h.verifySdkCallMade(t)
	})
}

func TestBenchmarkToken_TransferLargerThanAvailableFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		callerAddress := builders.ClientAddressForEd25519SignerForTests(0)
		targetAddress := builders.ClientAddressForEd25519SignerForTests(1)
		const amount, callerBalance = uint64(9999), uint64(20)

		t.Log("Runs BenchmarkToken.transfer large amount")

		call := processCallInput().WithMethod("BenchmarkToken", "transfer").WithArgs(amount, []byte(targetAddress)).Build()
		h.expectSdkCallMadeWithAddressGetCaller(callerAddress)
		h.expectSdkCallMadeWithStateRead(callerAddress, uint64ToBytes(callerBalance))

		output, err := h.service.ProcessCall(ctx, call)
		require.Error(t, err, "call should fail")
		require.Equal(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, output.CallResult, "call result should be smart contract error")
	})
}
