// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFileSystemBlockPersistence_RecoverFromPartiallyWrittenBlockRecord(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	ctrlRand := rand.NewControlledRand(t)

	conf := newTempFileConfig()
	defer conf.cleanDir()

	blocks := writeRandomBlocksToFile(t, conf, 2, ctrlRand)
	originalFileSize := getFileSize(t, conf)

	truncateFile(t, conf, originalFileSize-(ctrlRand.Int63n(30)+1)) // cut some bytes from end of file

	fsa, closeAdapter, err := NewFilesystemAdapterDriver(log.DefaultTestingLoggerAllowingErrors(t, "built index, found and ignoring invalid block records"), conf)
	require.NoError(t, err)
	defer closeAdapter()

	topBlockHeight, err := fsa.GetLastBlockHeight()
	require.NoError(t, err)

	require.EqualValues(t, 1, topBlockHeight, "expected partially written block record to be ignored")

	fsa.WriteNextBlock(blocks[1]) // re-append lost block
	recoveredFileSize := getFileSize(t, conf)

	require.Equal(t, originalFileSize, recoveredFileSize, "appending to a file with partial block record should occur at the end of last full record")
}

func TestFileSystemBlockPersistence_DataCorruption(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	ctrlRand := rand.NewControlledRand(t)

	conf := newTempFileConfig()
	defer conf.cleanDir()

	blocks := writeRandomBlocksToFile(t, conf, 2, ctrlRand)

	blocksFileSize1 := getFileSize(t, conf)
	flipBitInFile(t, conf, blocksFileSize1-(ctrlRand.Int63n(100)+1), byte(1)<<uint(ctrlRand.Intn(8))) // flip 1 bit in last block record

	fsa, closeAdapter, err := NewFilesystemAdapterDriver(log.DefaultTestingLoggerAllowingErrors(t, "built index, found and ignoring invalid block records"), conf)
	require.NoError(t, err)
	defer closeAdapter()

	topBlockHeight, err := fsa.GetLastBlockHeight()
	require.NoError(t, err)
	require.EqualValues(t, 1, topBlockHeight, "expected corrupt block record to be ignored")

	fsa.WriteNextBlock(blocks[1]) // re-append lost block

	blocksFileSize2 := getFileSize(t, conf)
	require.Equal(t, blocksFileSize1, blocksFileSize2, "appending to a file with partial block record should occur at the end of last valid record")
}
