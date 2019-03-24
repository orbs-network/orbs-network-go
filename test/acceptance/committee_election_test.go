// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build unsafetests

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("make sure it arrived to non-elected")

			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 10, token.GetBalance(ctx, 4, 6))

			t.Log("send transaction to one of the non-elected")

			_, txHash = token.Transfer(ctx, 4, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 20, token.GetBalance(ctx, 4, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 4, []int{0, 1, 2, 3})

			t.Log("make sure it arrived to elected")

			network.WaitForTransactionInNodeState(ctx, txHash, 2)
			require.EqualValues(t, 20, token.GetBalance(ctx, 2, 6))

			t.Log("test done, shutting down")

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
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 3, []int{2, 3, 4, 5})

			t.Log("test done, shutting down")

		})
}

func TestLeanHelix_AllNodesLoseElectionButReturn(t *testing.T) {
	newHarness().
		WithNumNodes(8).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")

			response, _ := contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to the first group")

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("elect 4,5,6,7 - entire first group loses")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 4, []int{4, 5, 6, 7})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to the first group after loss")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 20, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{4, 5, 6, 7})

			t.Log("elect 0,1,2,3 - first group returns")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction to the first node after return")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 30, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("test done, shutting down")

		})
}

func TestLeanHelix_GrowingElectedAmount(t *testing.T) {
	newHarness().
		WithNumNodes(7).
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
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("elect 0,1,2,3,4,5,6")

			response, _ = contract.UnsafeTests_SetElectedValidators(ctx, 3, []int{0, 1, 2, 3, 4, 5, 6})
			require.Equal(t, response.TransactionReceipt().ExecutionResult(), protocol.EXECUTION_RESULT_SUCCESS)

			t.Log("send transaction")

			_, txHash = token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 20, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3, 4, 5, 6})

			t.Log("test done, shutting down")

		})
}

func verifyTxSignersAreFromGroup(t testing.TB, ctx context.Context, api callcontract.CallContractAPI, txHash primitives.Sha256, nodeIndex int, allowedIndexes []int) {
	response := api.GetTransactionReceiptProof(ctx, txHash, nodeIndex)
	signers, err := digest.GetBlockSignersFromReceiptProof(response.PackedProof())
	require.NoError(t, err)
	signerIndexes := testKeys.NodeAddressesForTestsToIndexes(signers)
	t.Logf("signers of txHash %s are %v", txHash, signerIndexes)
	require.Subset(t, allowedIndexes, signerIndexes, "tx signers should be subset of allowed group")
}
