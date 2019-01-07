package test

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersistenceAdapter_CanAccessBlocksOutOfOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	ctrlRand := test.NewControlledRand(t)

	conf := newTempFileConfig()
	defer conf.cleanDir()

	blocks := writeBlocksToFile(t, conf, 50, ctrlRand)

	fsa, closeAdapter, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)
	defer closeAdapter()

	// read each block out of order
	for _, i := range ctrlRand.Perm(len(blocks)) {
		h := primitives.BlockHeight(i + 1)
		err := fsa.ScanBlocks(h, 1, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
			test.RequireCmpEqual(t, blocks[i], page[0], "expected to succeed in reading block at height %v", h)
			t.Logf("successfully read block height %v", i+1)
			return false
		})
		require.NoError(t, err)
	}
}
