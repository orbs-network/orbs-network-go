package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_, dirToCleanup := getProcessorArtifactPath()
	os.RemoveAll(dirToCleanup)

	exitCode := m.Run()

	os.RemoveAll(dirToCleanup)
	os.Exit(exitCode)
}
