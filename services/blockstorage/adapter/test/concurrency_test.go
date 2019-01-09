package test

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCanWriteAndScanConcurrently(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	ctrlRand := test.NewControlledRand(t)
	blocks := builders.RandomizedBlockChain(2, ctrlRand)

	conf := newTempFileConfig()
	defer conf.cleanDir()

	fsa, closeAdapter, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)
	defer closeAdapter()

	err = fsa.WriteNextBlock(blocks[0]) // write only the first block in the chain
	require.NoError(t, err)

	var topHeightRead primitives.BlockHeight
	secondBlockWritten, midScan, finishedScan := newSignalChan(), newSignalChan(), newSignalChan()
	go func() {
		err := fsa.ScanBlocks(1, 1, func(height primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
			if height == 1 {
				signal(midScan)
			}
			waitFor(secondBlockWritten)
			topHeightRead = height
			return true
		})
		require.NoError(t, err, "expected scan to complete with no error")
		signal(finishedScan)
	}()

	waitFor(midScan)

	err = fsa.WriteNextBlock(blocks[1]) // write the second block while a block scan is ongoing
	require.NoError(t, err, "should be able to write block while scanning")

	signal(secondBlockWritten)

	waitFor(finishedScan)

	require.EqualValues(t, 2, topHeightRead, "expected a block scan which began before the last write operation to return the last block written")
}

func newSignalChan() chan struct{} {
	return make(chan struct{})
}

func signal(ch chan struct{}) {
	close(ch)
}

func waitFor(ch chan struct{}) {
	<-ch
}
