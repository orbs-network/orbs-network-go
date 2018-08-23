package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"testing"
	"time"
)

func TestSyncHandleBlockAvailabilityRequest(t *testing.T) {
	driver := NewDriver()

	driver.blockSync.When("SendBlockAvailabilityResponse", mock.Any).Times(1)

	message := &gossipmessages.BlockAvailabilityRequestMessage{
		SignedRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			LastCommittedBlockHeight: primitives.BlockHeight(0),
		}).Build(),
	}
	input := &gossiptopics.BlockAvailabilityRequestInput{
		Message: message,
	}

	driver.expectCommitStateDiffTimes(2)

	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).WithBlockCreated(time.Now()).Build())
	driver.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(2)).WithBlockCreated(time.Now()).Build())

	driver.blockStorage.HandleBlockAvailabilityRequest(input)

	driver.verifyMocks(t)
}
