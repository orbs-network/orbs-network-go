package test

import (
	"github.com/orbs-network/orbs-network-go/config"
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

	waitFunc1 := createAdapterAndSleepInChildProcess(t, c.BlockStorageDataDir())
	waitFunc2 := createAdapterAndSleepInChildProcess(t, c.BlockStorageDataDir())

	requireOneFailOnePass(t, waitFunc1(), waitFunc2())

	waitFunc3 := createAdapterAndSleepInChildProcess(t, c.BlockStorageDataDir())
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

	err := lockAndRelease(c)
	require.NoError(t, err, "should succeed in creating an adapter for a non-existing temp file")

	time.Sleep(500 * time.Millisecond)

	err = lockAndRelease(c)
	require.NoError(t, err, "should succeed in creating a second adapter for same file after closing first adapter")
}

func lockAndRelease(c config.FilesystemBlockPersistenceConfig) error {
	_, cancel, err := NewFilesystemAdapterDriver(c)
	if err != nil {
		return err
	}
	cancel()
	return nil
}
