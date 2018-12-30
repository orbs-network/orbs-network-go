package test

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
	"time"
)

func TestAdvisoryLock_ExclusiveLock_ConcurrentProcesses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}

	waitFunc1 := lockAndWaitInChildProcess(t)

	time.Sleep(200 * time.Millisecond) // give the first process a head start

	waitFunc2 := lockAndWaitInChildProcess(t)

	require.NoError(t, waitFunc1(), "expected first process to lock successfully")
	require.EqualError(t, waitFunc2(), "exit status 1", "expected accessing same file from different processes to fail %v", time.Now())
}

func lockAndWaitInChildProcess(t *testing.T) func() error {
	cmd := exec.Command("go", "test", "-run", "TestAdvisoryLock_ExclusiveLock_LockAndWaitForCollision")
	err := cmd.Start()
	require.NoError(t, err)
	return func() error {
		return cmd.Wait()
	}
}

// invoked indirectly by lockAndWaitInChildProcess()
func TestAdvisoryLock_ExclusiveLock_LockAndWaitForCollision(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}

	c := newFixedPathConfig()
	defer c.cleanDir()

	release, err := lock(c)
	require.NoError(t, err)

	defer release()
	time.Sleep(1 * time.Second)
}

func TestAdvisoryLock_CanReleaseLock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}
	c := newTempFileConfig()
	defer c.cleanDir()

	err := lockAndRelease(c)
	require.NoError(t, err)

	err = lockAndRelease(c)
	require.NoError(t, err)
}

func lock(c config.FilesystemBlockPersistenceConfig) (func(), error) {
	_, cancel, err := NewFilesystemAdapterDriver(c)
	if err != nil {
		return nil, err
	}
	return cancel, nil
}

func lockAndRelease(c config.FilesystemBlockPersistenceConfig) error {
	_, cancel, err := NewFilesystemAdapterDriver(c)
	if err != nil {
		return err
	}
	cancel()
	return nil
}

func newFixedPathConfig() *localConfig {
	return &localConfig{
		dir: "/tmp/orbs/tests/block_storage_adapter/advisory_lock_test",
	}
}
