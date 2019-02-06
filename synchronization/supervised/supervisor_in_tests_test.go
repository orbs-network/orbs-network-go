package supervised

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"testing"
	"time"
)

// this is a manual test, could not find an easy way to automate it
// uncomment it to test it out

func TestGoOnce_FailsTestOnPanic(t *testing.T) {
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

func theFunctionThrowingThePanic() {
	panic("oh no")
}
