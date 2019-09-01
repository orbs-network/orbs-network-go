// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const blocksFilename = "blocks"

func NewFilesystemAdapterDriver(logger log.Logger, conf config.FilesystemBlockPersistenceConfig) (adapter.BlockPersistence, func(), error) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	persistence, err := filesystem.NewBlockPersistence(ctx, conf, logger, metric.NewRegistry())
	if err != nil {
		return nil, nil, err
	}

	closeAdapter := func() {
		cancelCtx()
		time.Sleep(500 * time.Millisecond) // time to release any lock
	}

	return persistence, closeAdapter, nil
}

type localConfig struct {
	dir         string
	chainId     primitives.VirtualChainId
	networkType protocol.SignerNetworkType
}

func newTempFileConfig() *localConfig {
	dirName, err := ioutil.TempDir("", "contract_test_block_persist")
	if err != nil {
		panic(err)
	}
	return &localConfig{
		dir:         dirName,
		chainId:     0xFF,
		networkType: protocol.NETWORK_TYPE_TEST_NET,
	}
}
func (l *localConfig) BlockStorageFileSystemDataDir() string {
	return l.dir
}

func (l *localConfig) BlockStorageFileSystemMaxBlockSizeInBytes() uint32 {
	return 64 * 1024 * 1024
}

func (l *localConfig) VirtualChainId() primitives.VirtualChainId {
	return l.chainId
}

func (l *localConfig) NetworkType() protocol.SignerNetworkType {
	return l.networkType
}

func (l *localConfig) cleanDir() {
	_ = os.RemoveAll(l.BlockStorageFileSystemDataDir()) // ignore errors - nothing to do
}

func (l *localConfig) setVirtualChainId(value primitives.VirtualChainId) {
	l.chainId = value
}

func (l *localConfig) setNetworkType(value protocol.SignerNetworkType) {
	l.networkType = value
}

func getFileSize(t *testing.T, conf *localConfig) int64 {
	blocksFile, err := os.Open(filepath.Join(conf.BlockStorageFileSystemDataDir(), blocksFilename))
	require.NoError(t, err)
	blocksFileInfo2, err := blocksFile.Stat()
	require.NoError(t, err)
	err = blocksFile.Close()
	require.NoError(t, err)
	return blocksFileInfo2.Size()
}

func truncateFile(t *testing.T, conf *localConfig, size int64) {
	blocksFile, err := os.OpenFile(filepath.Join(conf.BlockStorageFileSystemDataDir(), blocksFilename), os.O_RDWR, 0666)
	require.NoError(t, err)
	err = blocksFile.Truncate(size)
	require.NoError(t, err)
	err = blocksFile.Close()
	require.NoError(t, err)
}

func flipBitInFile(t *testing.T, conf *localConfig, offset int64, bitMask byte) {
	blocksFile, err := os.OpenFile(filepath.Join(conf.BlockStorageFileSystemDataDir(), blocksFilename), os.O_RDWR, 0666)
	require.NoError(t, err)
	b := make([]byte, 1)
	n, err := blocksFile.ReadAt(b, offset)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	b[0] = b[0] ^ bitMask
	n, err = blocksFile.WriteAt(b, offset)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	err = blocksFile.Close()
	require.NoError(t, err)
}

func writeRandomBlocksToFile(t *testing.T, logger log.Logger, conf *localConfig, numBlocks int32, ctrlRand *rand.ControlledRand) []*protocol.BlockPairContainer {
	fsa, closeAdapter, err := NewFilesystemAdapterDriver(logger, conf)
	require.NoError(t, err)
	defer closeAdapter()

	blockChain := builders.RandomizedBlockChain(numBlocks, ctrlRand)

	for _, block := range blockChain {
		_, _, err = fsa.WriteNextBlock(block)
		require.NoError(t, err)
	}
	return blockChain
}
