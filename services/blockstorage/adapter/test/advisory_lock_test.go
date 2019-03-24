// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAdvisoryLock_AdapterTakesExclusiveLock_ConcurrentProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}

	c := newTempFileConfig()
	defer c.cleanDir()

	waitFunc1 := createAdapterAndSleepInChildProcess(t, c.BlockStorageFileSystemDataDir())
	waitFunc2 := createAdapterAndSleepInChildProcess(t, c.BlockStorageFileSystemDataDir())

	requireOneFailOnePass(t, waitFunc1(), waitFunc2())

	waitFunc3 := createAdapterAndSleepInChildProcess(t, c.BlockStorageFileSystemDataDir())
	require.NoError(t, waitFunc3(), "after locking process shuts down subsequent attempts should succeed")
}

func requireOneFailOnePass(t *testing.T, err1 error, err2 error) {
	if (err1 == nil) == (err2 == nil) {
		if err1 == nil {
			t.Fatal("expected one process to initialize successfully, and the other to fail. both succeeded.")
		} else {
			t.Fatalf("expected one process to initialize successfully, and the other to fail. both failed. err1: %s, err2: %s", err1, err2)
		}
	}
}

func createAdapterAndSleepInChildProcess(t *testing.T, dir string) func() error {
	cmd := exec.Command("go", "run", filepath.Join(config.GetCurrentSourceFileDirPath(), "main", "create_adapter_and_sleep_main.go"), dir)
	err := cmd.Start()
	require.NoError(t, err)
	return func() error {
		return cmd.Wait()
	}
}

func TestAdvisoryLock_AdapterCanReleaseLock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}

	c := newTempFileConfig()
	defer c.cleanDir()

	err := lockAndRelease(t, c)
	require.NoError(t, err, "should succeed in creating an adapter for a non-existing temp file")

	time.Sleep(500 * time.Millisecond)

	err = lockAndRelease(t, c)
	require.NoError(t, err, "should succeed in creating a second adapter for same file after closing first adapter")
}

func lockAndRelease(tb testing.TB, c config.FilesystemBlockPersistenceConfig) error {
	_, cancel, err := NewFilesystemAdapterDriver(log.DefaultTestingLogger(tb), c)
	if err != nil {
		return err
	}
	cancel()
	return nil
}
