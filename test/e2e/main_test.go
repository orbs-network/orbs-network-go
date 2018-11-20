package e2e

import (
	"os"
	"testing"
)

func runOnce(m *testing.M) int {
	return m.Run()
}

func runTwice(m *testing.M) int {
	if exitCode := runOnce(m); exitCode != 0 {
		return exitCode
	}

	return runOnce(m)
}

func TestMain(m *testing.M) {
	exitCode := 0

	bootstrap := getConfig().bootstrap

	if bootstrap {
		cleanNativeProcessorCache()
		n := newInProcessE2ENetwork()

		exitCode = runTwice(m)
		n.gracefulShutdown()

		cleanNativeProcessorCache()
	} else {
		exitCode = runOnce(m)
	}

	os.Exit(exitCode)
}

func cleanNativeProcessorCache() {
	_, dirToCleanup := getProcessorArtifactPath()
	os.RemoveAll(dirToCleanup)
}
