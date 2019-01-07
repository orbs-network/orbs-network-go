package test

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
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

	blocks := writeRandomBlocksToFile(t, conf, 50, ctrlRand)

	fsa, closeAdapter, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)
	defer closeAdapter()

	for _, i := range ctrlRand.Perm(len(blocks)) { // read each block out of order
		h := primitives.BlockHeight(i + 1)
		block, err := readOneBlock(fsa, h)
		test.RequireCmpEqual(t, blocks[i], block, "expected to succeed in reading block at height %v", h)
		t.Logf("successfully read block height %v", i+1)
		require.NoError(t, err)
	}
}

func readOneBlock(fsa adapter.BlockPersistence, h primitives.BlockHeight) (*protocol.BlockPairContainer, error) {
	var block *protocol.BlockPairContainer
	err := fsa.ScanBlocks(h, 1, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
		block = page[0]
		return false
	})
	return block, err
}
