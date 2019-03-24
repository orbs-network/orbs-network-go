// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeployNativeContract(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM
		network.MockContract(contracts.MockForCounter(), string(contracts.NativeSourceCodeForCounter(counterStart)))
		contract := callcontract.NewContractClient(network)

		t.Log("deploying contract")

		response, txHash := contract.DeployNativeCounterContract(ctx, 1, 0)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

		t.Log("wait for node to sync with deployment")
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		require.EqualValues(t, counterStart, contract.CounterGet(ctx, 0), "get counter after deploy")

		t.Log("transacting with contract")

		_, txHash = contract.CounterAdd(ctx, 1, 17)

		t.Log("wait for node to sync with transaction")
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		require.EqualValues(t, counterStart+17, contract.CounterGet(ctx, 0), "get counter after transaction")

	})
}

func TestLockNativeDeployment(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {

		counterStart := contracts.MOCK_COUNTER_CONTRACT_START_FROM
		network.MockContract(contracts.MockForCounter(), string(contracts.NativeSourceCodeForCounter(counterStart)))
		contract := callcontract.NewContractClient(network)

		t.Log("lock native deployment to account 5 should succeed")

		response, _ := contract.LockNativeDeployment(ctx, 0, 5)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

		t.Log("lock native deployment to account 6 should fail (already locked)")

		response, _ = contract.LockNativeDeployment(ctx, 0, 6)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT)

		t.Log("deploy native contract from account 6 should fail")

		response, txHash := contract.DeployNativeCounterContract(ctx, 0, 6)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT)
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		t.Log("transacting with contract should fail")

		response, _ = contract.CounterAdd(ctx, 0, 17)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED)

		t.Log("unlock native deployment from account 5 should succeed")

		response, _ = contract.UnlockNativeDeployment(ctx, 0, 5)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

		t.Log("lock native deployment to account 6 should succeed")

		response, _ = contract.LockNativeDeployment(ctx, 0, 6)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

		t.Log("deploy native contract from account 6 should succeed")

		response, txHash = contract.DeployNativeCounterContract(ctx, 0, 6)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)
		network.WaitForTransactionInNodeState(ctx, txHash, 0)

		t.Log("transacting with contract should succeed")

		response, _ = contract.CounterAdd(ctx, 0, 17)
		require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

	})
}
