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

	// write one block
	err = fsa.WriteNextBlock(blocks[0])
	require.NoError(t, err)

	// scan blocks - signal after first block received, pause and wait for signal to continue, then signal when finish
	topHeightRead := 0
	wroteNextBlockSignal, midScanSignal, finishedScanSignal := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() {
		err := fsa.ScanBlocks(1, 1, func(first primitives.BlockHeight, page []*protocol.BlockPairContainer) (wantsMore bool) {
			if first == 1 {
				close(midScanSignal)
			}
			<-wroteNextBlockSignal
			topHeightRead = int(first)
			return true
		})
		require.NoError(t, err)
		close(finishedScanSignal)
	}()

	// wait for scan to begin
	<-midScanSignal

	// write another block while scanning is ongoing
	err = fsa.WriteNextBlock(blocks[1])
	require.NoError(t, err, "should be able to write block while scanning")

	// signal scan to proceed
	close(wroteNextBlockSignal)

	// wait for scan to finish
	<-finishedScanSignal

	require.EqualValues(t, 2, topHeightRead)
}
