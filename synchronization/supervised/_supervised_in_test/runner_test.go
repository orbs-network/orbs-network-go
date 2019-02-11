package _supervised_in_test

import (
	"testing"
	"time"

	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	logLine "github.com/orbs-network/orbs-network-go/synchronization/supervised/test"
)

func TestGoOnce_FailsTestOnPanicAndPrintsLogs(t *testing.T) {
	t.Log(logLine.BeforeLoggerCreated)
	testLogger := log.DefaultTestingLogger(t)
	testLogger.Info(logLine.LoggedWithLogger)

	supervised.GoOnce(testLogger, func() {
		t.Log(logLine.BeforeCallPanic)
		theFunctionThrowingThePanic()
		t.Log(logLine.AfterCallPanic)
	})

	time.Sleep(100 * time.Millisecond)

	testLogger.Info(logLine.MustNotShow)
	t.Log(logLine.MustShow)
}

func TestTRun_FailsTestOnPanicAndPrintsLogs(t *testing.T) {
	t.Log(logLine.ParentScopeBeforeTest)

	// t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside
	t.Run("SubTest", func(t *testing.T) {

		t.Log(logLine.BeforeLoggerCreated)
		subTestLogger := log.DefaultTestingLogger(t)
		subTestLogger.Info(logLine.LoggedWithLogger)

		supervised.Recover(subTestLogger, func() {

			t.Log(logLine.BeforeCallPanic)
			theFunctionThrowingThePanic()
			t.Log(logLine.AfterCallPanic)

		})
	})

	t.Log(logLine.ParentScopeAfterTest)
}

func theFunctionThrowingThePanic() {
	panic("oh no")
}
