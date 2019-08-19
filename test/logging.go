package test

import (
	"github.com/orbs-network/scribe/log"
	"testing"
)

type LoggingHarness struct {
	Logger     log.Logger
	testOutput *log.TestOutput
	T          testing.TB
}

func (h *LoggingHarness) AllowErrorsMatching(pattern string) {
	h.testOutput.AllowErrorsMatching(pattern)
}

func WithLogger(tb testing.TB, f func(harness *LoggingHarness)) {
	testOutput := log.NewTestOutput(tb, log.NewHumanReadableFormatter())
	h := &LoggingHarness{
		Logger:     log.GetLogger().WithOutput(testOutput),
		testOutput: testOutput,
		T:          tb,
	}
	defer testOutput.TestTerminated()
	f(h)
	RequireNoUnexpectedErrors(tb, testOutput)
}
