package with

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/scribe/log"
	"testing"
	"time"
)

type ConcurrencyHarness struct {
	govnr.TreeSupervisor
	Logger     log.Logger
	testOutput *log.TestOutput
	T          testing.TB
}

func (h *ConcurrencyHarness) AllowErrorsMatching(pattern string) {
	h.testOutput.AllowErrorsMatching(pattern)
}

// creates a harness that should be used for running concurrent tests; these are tests that start long-running goroutines that need supervision
// the test function will run inside this function, and following the test running a synchronized shutdown of the supervised SUT will take place,
// followed by an assertion that no unexpected errors have been logged
func Concurrency(tb testing.TB, f func(ctx context.Context, harness *ConcurrencyHarness)) {
	testOutput := log.NewTestOutput(tb, log.NewHumanReadableFormatter())
	h := &ConcurrencyHarness{
		Logger:     log.GetLogger().WithOutput(testOutput),
		testOutput: testOutput,
		T:          tb,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer shutdown(h)
	defer cancel()
	defer testOutput.TestTerminated()
	f(ctx, h)
	RequireNoUnexpectedErrors(tb, testOutput)
}

func shutdown(waiter govnr.ShutdownWaiter) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	waiter.WaitUntilShutdown(ctx)
}
