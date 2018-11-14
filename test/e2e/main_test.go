package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	cleanNativeProcessorCache()

	n := newInProcessE2ENetwork()

	exitCode := m.Run()

	n.gracefulShutdown()

	cleanNativeProcessorCache()
	os.Exit(exitCode)
}

func cleanNativeProcessorCache() {
	_, dirToCleanup := getProcessorArtifactPath()
	os.RemoveAll(dirToCleanup)
}
