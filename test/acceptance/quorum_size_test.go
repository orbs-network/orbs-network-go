// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNetworkStartedWithEnoughNodes_SucceedsClosingBlocks_BenchmarkConsensus(t *testing.T) {
	newHarness().
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		WithNumNodes(6).
		WithNumRunningNodes(4).
		WithRequiredQuorumPercentage(66).
		WithLogFilters(
			log.ExcludeEntryPoint("BlockSync"),
			log.IgnoreMessagesMatching("Metric recorded"),
			log.ExcludeEntryPoint("LeanHelixConsensus")).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := network.DeployBenchmarkTokenContract(ctx, 5)

			out, _ := contract.Transfer(ctx, 0, uint64(23), 5, 6)
			require.NotNil(t, out)
		})
}
