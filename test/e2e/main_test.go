package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	cleanNativeProcessorCache()

	exitCode := m.Run()

	cleanNativeProcessorCache()
	os.Exit(exitCode)
}

func cleanNativeProcessorCache() {
	_, dirToCleanup := getProcessorArtifactPath()
	os.RemoveAll(dirToCleanup)
}
