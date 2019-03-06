// +build unsafetests

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeanHelix_CommitTransactionToElected(t *testing.T) {
	newHarness().
		WithNumNodes(6).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect first 4 out of 6")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 0, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to one of the elected")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))

			t.Log("make sure it arrived to non-elected")

			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 10, token.GetBalance(ctx, 4, 6))

			t.Log("send transaction to one of the non-elected")

			_, txHash = token.Transfer(ctx, 4, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 20, token.GetBalance(ctx, 4, 6))

			t.Log("make sure it arrived to elected")

			network.WaitForTransactionInNodeState(ctx, txHash, 2)
			require.EqualValues(t, 20, token.GetBalance(ctx, 2, 6))

		})
}

func TestLeanHelix_MultipleReElections(t *testing.T) {
	newHarness().
		WithNumNodes(6).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("elect 1,2,3,4")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{1, 2, 3, 4})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("elect 2,3,4,5")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{2, 3, 4, 5})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to one of the elected")

			_, txHash := token.Transfer(ctx, 3, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 3)
			require.EqualValues(t, 10, token.GetBalance(ctx, 3, 6))

		})
}

func TestLeanHelix_NodeLosesElectionButReturns(t *testing.T) {
	newHarness().
		WithNumNodes(6).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to the first node")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))

			t.Log("elect 1,2,3,4 - first node loses")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{1, 2, 3, 4})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to the first node after loss")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 20, token.GetBalance(ctx, 0, 6))

			t.Log("elect 0,1,2,3 - first node returns")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to the first node after return")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 30, token.GetBalance(ctx, 0, 6))

		})
}

func TestLeanHelix_GrowingElectedAmount(t *testing.T) {
	newHarness().
		WithNumNodes(6).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))

			t.Log("elect 0,1,2,3,4,5")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3, 4, 5})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 20, token.GetBalance(ctx, 0, 6))

		})
}
