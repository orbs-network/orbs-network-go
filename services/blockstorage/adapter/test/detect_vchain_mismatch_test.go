package test

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersistenceAdapter_DetectsVirtualChainMismatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Integration tests in short mode")
	}

	conf := newTempFileConfig()
	defer conf.cleanDir()

	writeRandomBlocksToFile(t, conf, 1, test.NewControlledRand(t))

	conf.setVirtualChainId(conf.VirtualChainId() + 1)

	_, _, err := NewFilesystemAdapterDriver(log.DefaultTestingLogger(t), conf)
	require.Error(t, err, "expected error when trying to open a blocks file from a different virtual chain")
}
