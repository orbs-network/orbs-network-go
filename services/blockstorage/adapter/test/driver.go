package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"io/ioutil"
	"os"
)

// TODO V1 check that we can read concurrently from different places in the file
// TODO V1 check that we don't use long locks - that concurrent reads don't wait on each other
// TODO V1 init flow - build indexes
// TODO V1 error during persistence
// TODO V1 tampering FS?
// TODO V1 checks and validations
// TODO V1 codec version
// TODO V1 file format header and version
// TODO V1 test that if writing a block while scanning is ongoing we will receive the new
// TODO V1 write test for recovering from a corrupt writing file handle
// TODO V1 file format includes a version, and if the version not supported don't run

func NewFilesystemAdapterDriverWithTempFile() (adapter.BlockPersistence, func(), error) {
	conf := newTempFileConfig()
	persistence, closeAdapter, err := NewFilesystemAdapterDriver(conf)
	if err != nil {
		return nil, nil, err
	}

	closeAdapterAndCleanFiles := func() {
		closeAdapter()
		conf.cleanDir()
	}

	return persistence, closeAdapterAndCleanFiles, nil
}

func NewFilesystemAdapterDriver(conf config.FilesystemBlockPersistenceConfig) (adapter.BlockPersistence, func(), error) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	persistence, err := adapter.NewFilesystemBlockPersistence(ctx, conf, log.GetLogger(), metric.NewRegistry())
	if err != nil {
		return nil, nil, err
	}

	closeAdapter := func() {
		cancelCtx()
	}

	return persistence, closeAdapter, nil
}

type localConfig struct {
	dir string
}

func newTempFileConfig() *localConfig {
	dirName, err := ioutil.TempDir("", "contract_test_block_persist")
	if err != nil {
		panic(err)
	}
	return &localConfig{
		dir: dirName,
	}
}
func (l *localConfig) BlockStorageDataDir() string {
	return l.dir
}

func (l *localConfig) cleanDir() {
	_ = os.RemoveAll(l.BlockStorageDataDir()) // ignore errors - nothing to do
}
