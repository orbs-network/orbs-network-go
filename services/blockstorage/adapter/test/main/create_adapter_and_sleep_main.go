package main

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"os"
	"time"
)

func main() {
	release, err := createLockingAdapter()
	if err != nil {
		os.Exit(1)
	}

	defer release()
	time.Sleep(1 * time.Second)
}

func createLockingAdapter() (func(), error) {
	dir := os.Args[1]
	c := &localConfig{
		dir: dir,
	}
	_, cancel, err := test.NewFilesystemAdapterDriver(c)
	if err != nil {
		return nil, err
	}
	return cancel, nil
}

type localConfig struct {
	dir          string
	maxBlockSize uint32
}

func (l *localConfig) BlockStorageDataDir() string {
	return l.dir
}

func (l *localConfig) BlockStorageMaxBlockSize() uint32 {
	return 1024
}

func (l *localConfig) VirtualChainId() primitives.VirtualChainId {
	return 0xFF
}
