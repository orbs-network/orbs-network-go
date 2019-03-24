// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build unsafetests

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSubscriptionProblemThanBecomesOkAgain(t *testing.T) {
	newHarness().
		AllowingErrors("error validating transaction for preorder").
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("subscription problem")

			response, _ := contract.UnsafeTests_SetSubscriptionProblem(ctx, 0)
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction should fail")

			response, _ = token.Transfer(ctx, 0, 17, 5, 6)
			require.Equal(t, response.TransactionStatus(), protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER)
			require.EqualValues(t, 0, token.GetBalance(ctx, 0, 6))

			t.Log("subscription ok")

			response, _ = contract.UnsafeTests_SetSubscriptionOk(ctx, 0)
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction should succeed")

			response, txHash := token.Transfer(ctx, 0, 17, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.Equal(t, response.TransactionStatus(), protocol.TRANSACTION_STATUS_COMMITTED)
			require.EqualValues(t, 17, token.GetBalance(ctx, 0, 6))

		})
}
