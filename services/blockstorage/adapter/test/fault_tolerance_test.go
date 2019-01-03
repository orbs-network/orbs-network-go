package test

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFileSystemBlockPersistence_CrashDuringWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	ctrlRand := test.NewControlledRand(t)

	conf := newTempFileConfig()
	defer conf.cleanDir()

	blocks := writeBlocksToFile(t, conf, 2, ctrlRand)

	// cut bytes
	blocksFileSize1 := getFileSize(t, conf)
	truncateFile(t, conf, blocksFileSize1-(ctrlRand.Int63n(30)+1))

	// load new adapter
	fsa, closeAdapter, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)
	defer closeAdapter()

	// check block height
	topBlockHeight, err := fsa.GetLastBlockHeight()
	require.NoError(t, err)
	require.EqualValues(t, 1, topBlockHeight, "expected partial block to be ignored.")

	// append block
	fsa.WriteNextBlock(blocks[1])
	closeAdapter()

	blocksFileSize2 := getFileSize(t, conf)
	require.Equal(t, blocksFileSize1, blocksFileSize2, "appending should continue after last full block")
}

func TestFileSystemBlockPersistence_DataCorruption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	ctrlRand := test.NewControlledRand(t)

	conf := newTempFileConfig()
	defer conf.cleanDir()

	blocks := writeBlocksToFile(t, conf, 2, ctrlRand)

	// flip 1 bit in last block
	blocksFileSize1 := getFileSize(t, conf)
	flipBitInFile(t, conf, blocksFileSize1-(ctrlRand.Int63n(100)+1), byte(1)<<uint(ctrlRand.Intn(8)))

	// load new adapter
	fsa, closeAdapter, err := NewFilesystemAdapterDriver(conf)
	require.NoError(t, err)
	defer closeAdapter()

	// check block height
	topBlockHeight, err := fsa.GetLastBlockHeight()
	require.NoError(t, err)
	require.EqualValues(t, 1, topBlockHeight, "expected corrupt block to be ignored.")

	// append block
	fsa.WriteNextBlock(blocks[1])
	closeAdapter()

	blocksFileSize2 := getFileSize(t, conf)
	require.Equal(t, blocksFileSize1, blocksFileSize2, "appending should continue after last valid block")
}
