package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"testing"
)

func TestNonLeaderDoesNotCreateBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.consensusContext.Reset().When("RequestNewTransactionsBlock", mock.Any).Return(nil, nil).Times(0)
		h.createService(ctx)
		err := test.ConsistentlyVerify(h.consensusContext)
		if err != nil {
			t.Fatal("Did create block with ConsensusContext:", err)
		}
	})
}

func TestNonLeaderIgnoreFutureBlockHeight(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(false)
		h.createService(ctx)
		h.gossip.Reset().When("SendBenchmarkConsensusCommitted", mock.Any).Return(nil, nil).Times(0)
		h.service.HandleBenchmarkConsensusCommit(&gossiptopics.BenchmarkConsensusCommitInput{
			Message: &gossipmessages.BenchmarkConsensusCommitMessage{
				BlockPair: builders.BlockPair().WithHeight(1000).Build(),
			},
		})
		err := test.ConsistentlyVerify(h.gossip)
		if err != nil {
			t.Fatal("Did not ignore block with future block height:", err)
		}
	})
}
