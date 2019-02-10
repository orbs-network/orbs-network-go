package test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoOnce_FailsTestOnPanicAndPrintsLogs(t *testing.T) {
	expectedLogs := []string{
		Failed,
		BeforeLoggerCreated,
		LoggedWithLogger,
		BeforeCallPanic,
		MustShow,
	}
	unexpectedLogs := []string{
		AfterCallPanic,
		MustNotShow,
	}
	out, _ := exec.Command(
		"go",
		"test",
		"github.com/orbs-network/orbs-network-go/synchronization/supervised/_supervised_in_test/",
		"-run",
		"^(TestGoOnce_FailsTestOnPanicAndPrintsLogs)$").CombinedOutput()

	output := string(out)

	for _, logLine := range expectedLogs {
		require.Truef(t, strings.Contains(output, logLine), "log does not contain: %s", logLine)
	}
	for _, logLine := range unexpectedLogs {
		require.Falsef(t, strings.Contains(output, logLine), "log should not contain: %s", logLine)
	}

}

func TestTRun_FailsTestOnPanicAndPrintsLogs(t *testing.T) {
	expectedLogs := []string{
		Failed,
		ParentScopeBeforeTest,
		BeforeLoggerCreated,
		LoggedWithLogger,
		BeforeCallPanic,
		ParentScopeAfterTest,
	}
	unexpectedLogs := []string{
		AfterCallPanic,
	}
	out, _ := exec.Command(
		"go",
		"test",
		"github.com/orbs-network/orbs-network-go/synchronization/supervised/_supervised_in_test/",
		"-run",
		"^(TestTRun_FailsTestOnPanicAndPrintsLogs)$").CombinedOutput()

	output := string(out)

	for _, logLine := range expectedLogs {
		require.Truef(t, strings.Contains(output, logLine), "log does not contain: %s", logLine)
	}
	for _, logLine := range unexpectedLogs {
		require.Falsef(t, strings.Contains(output, logLine), "log should not contain: %s", logLine)
	}
}
