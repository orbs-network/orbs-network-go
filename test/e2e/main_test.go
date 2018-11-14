package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	cleanNativeProcessorCache()

	n := newInProcessE2ENetwork()

	m.Run()
	exitCode := m.Run() // run twice so that any test assuming a clean slate will fail; e2es shouldn't assume anything about system state

	n.gracefulShutdown()

	cleanNativeProcessorCache()
	os.Exit(exitCode)
}

func cleanNativeProcessorCache() {
	_, dirToCleanup := getProcessorArtifactPath()
	os.RemoveAll(dirToCleanup)
}
