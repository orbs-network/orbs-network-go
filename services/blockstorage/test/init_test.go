package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestInitSetsLastCommittedBlockHeightToZero(t *testing.T) {
	d := NewDriver()

	val, err := d.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})
	require.NoError(t, err)

	require.EqualValues(t, 0, val.LastCommittedBlockHeight)
	require.EqualValues(t, 0, val.LastCommittedBlockTimestamp)

	d.verifyMocks(t)
}

func TestInitSetsLastCommittedBlockHeightFromPersistence(t *testing.T) {
	now := time.Now()

	d := NewCustomSetupDriver(func(persistence adapter.InMemoryBlockPersistence, consensus *handlers.MockConsensusBlocksHandler) {
		for i := 1; i <= 10; i++ {
			now = now.Add(1 * time.Millisecond)
			persistence.WriteBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(i)).WithBlockCreated(now).Build())

		}

		out := &handlers.HandleBlockConsensusOutput{}

		consensus.When("HandleBlockConsensus", mock.Any).Return(out, nil).Times(1)
	})

	val, err := d.blockStorage.GetLastCommittedBlockHeight(&services.GetLastCommittedBlockHeightInput{})
	require.NoError(t, err)

	require.EqualValues(t, 10, val.LastCommittedBlockHeight)
	require.EqualValues(t, now.UnixNano(), val.LastCommittedBlockTimestamp)

	d.verifyMocks(t)
}
