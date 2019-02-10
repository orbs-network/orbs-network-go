package supervised

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"testing"
	"time"
)

// these are manual tests, could not find an easy way to automate them
// unskip them to test them out

func TestGoOnce_FailsTestOnPanicAndPrintsLogs(t *testing.T) {
	t.Skip("this test is designed to fail")

	t.Log("before logger is created")
	testLogger := log.DefaultTestingLogger(t)
	testLogger.Info("logged using the logger")

	GoOnce(testLogger, func() {
		t.Log("about to call theFunctionThrowingThePanic")
		theFunctionThrowingThePanic()
		t.Log("after call to theFunctionThrowingThePanic")
	})

	time.Sleep(100 * time.Millisecond)

	testLogger.Info("this is not supposed to show when the test fails")
	t.Log("this is supposed to show even if the test fails")
}

func TestTRun_FailsTestOnPanicAndPrintsLogs(t *testing.T) {
	t.Skip("this test is designed to fail")

	t.Log("this is in parent before the sub test")

	// t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside
	t.Run("SubTest", func(t *testing.T) {

		t.Log("before logger is created")
		subTestLogger := log.DefaultTestingLogger(t)
		subTestLogger.Info("logged using the logger")

		Recover(subTestLogger, func() {

			t.Log("about to call theFunctionThrowingThePanic")
			theFunctionThrowingThePanic()
			t.Log("after call to theFunctionThrowingThePanic")

		})
	})

	t.Log("this is parent after the sub test")
}

func theFunctionThrowingThePanic() {
	panic("oh no")
}
