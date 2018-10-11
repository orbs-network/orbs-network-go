package test

import (
	"context"
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSourceRespondToAvailabilityRequests(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)
		harness.setupSomeBlocks(3)
		harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, nil).Times(1)
		x := harness.getLastBlockHeight(t)
		harness.logger.Info("c", log.Stringable("bh", x.LastCommittedBlockHeight))
		msg := builders.BlockAvailabilityRequestInput().
			WithFirstBlockHeight(1).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(2)).
			Build()
		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(msg)

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any).Return(nil, nil).AtLeast(0)
		require.NoError(t, err, "expecting a happy flow")

		harness.verifyMocks(t, 1)
	})
}

func TestSourceRespondsNothingToAvailabilityRequestIfSourceIsBehindPetitioner(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		harness := newHarness(ctx)
		harness.expectCommitStateDiff()
		harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).Build())

		harness.gossip.Never("SendBlockAvailabilityResponse", mock.Any)

		msg := builders.BlockAvailabilityRequestInput().WithLastCommittedBlockHeight(primitives.BlockHeight(20)).Build()
		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(msg)

		require.NoError(t, err, "expecting a happy flow")
		harness.verifyMocks(t, 1)
	})
}

func TestSourceIgnoresSendBlockAvailabilityRequestsIfFailedToRespond(t *testing.T) {

	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)
		harness.setupSomeBlocks(3)

		harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)
		msg := builders.BlockAvailabilityRequestInput().
			WithFirstBlockHeight(1).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(2)).
			Build()
		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(msg)

		require.Error(t, err, "expecting an error from the server event flow")

		harness.verifyMocks(t, 1)
	})
}

func TestSourceRespondsWithChunks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)
		harness.setupSomeBlocks(12)

		msg := builders.BlockSyncRequestInput().
			WithFirstBlockHeight(primitives.BlockHeight(1)).
			Build()

		firstHeight := primitives.BlockHeight(1)
		lastHeight := primitives.BlockHeight(10)

		var blocks []*protocol.BlockPairContainer

		for i := firstHeight; i <= lastHeight; i++ {
			blocks = append(blocks, builders.BlockPair().WithHeight(i).Build())
		}

		// TODO: how do i get the arguments sent to the mock?
		harness.gossip.When("SendBlockSyncResponse", mock.Any).Return(nil, nil).Times(1)
		harness.blockStorage.HandleBlockSyncRequest(msg)
		harness.verifyMocks(t, 1)
	})
}

//
//func TestSourceAnyStateIgnoresBlockSyncRequestIfSourceIsBehindOrInSync(t *testing.T) {
//	firstHeight := primitives.BlockHeight(11)
//	lastHeight := primitives.BlockHeight(10)
//
//	event := builders.BlockSyncRequestInput().WithFirstBlockHeight(firstHeight).WithLastCommittedBlockHeight(lastHeight).Build().Message
//
//	for _, state := range allStates(true) {
//		t.Run("state="+blockSyncStateNameLookup[state], func(t *testing.T) {
//			harness := newBlockSyncHarness()
//
//			harness.storage.When("LastCommittedBlockHeight").Return(lastHeight).Times(1)
//			harness.storage.Never("GetBlocks")
//			harness.gossip.Never("SendBlockSyncResponse", mock.Any)
//
//			availabilityResponses := []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}
//
//			newState, availabilityResponses := harness.blockSync.transitionState(state, event, availabilityResponses, harness.startSyncTimer)
//
//			require.Equal(t, state, newState, "state change was not expected")
//			require.Equal(t, availabilityResponses, []*gossipmessages.BlockAvailabilityResponseMessage{nil, nil}, "availabilityResponses should remain the same")
//
//			harness.verifyMocks(t)
//		})
//	}
//}
