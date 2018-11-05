package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
)

func TestAvailabilityRequestWithNilOrInvalidPayloadDoesNotPanic(t *testing.T) {
	// TODO move the next ~10 lines to a harness if writing more tests
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	ctx := context.Background()
	gossip := &gossiptopics.MockBlockSync{}

	s := service{
		logger: logger,
		gossip: gossip,
	}

	s.lastBlockLock = &sync.RWMutex{}

	lastBlock := builders.BlockPair().WithHeight(10).Build()
	s.lastCommittedBlock = lastBlock
	input := builders.BlockAvailabilityRequestInput().WithFirstBlockHeight(5).Build()
	noPanicFunction := func() {
		s.sourceHandleBlockAvailabilityRequest(ctx, input.Message)
	}

	input.Message = nil
	require.NotPanics(t, noPanicFunction, "should not panic when message is nil")

	input = builders.BlockAvailabilityRequestInput().WithFirstBlockHeight(5).Build()
	input.Message.Sender = nil
	require.NotPanics(t, noPanicFunction, "should not panic when Sender inside message is nil")

	input = builders.BlockAvailabilityRequestInput().WithFirstBlockHeight(5).Build()
	input.Message.SignedBatchRange = nil
	require.NotPanics(t, noPanicFunction, "should not panic when SignedBatchRange inside message is nil")
}
