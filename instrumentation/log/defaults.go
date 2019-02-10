package log

import (
	"testing"
)

func DefaultTestingLogger(tb testing.TB) BasicLogger {
	return GetLogger().WithOutput(NewTestOutput(tb, NewHumanReadableFormatter()))
}

func DefaultTestingLoggerAllowingErrors(tb testing.TB, errorPattern string) BasicLogger {
	output := NewTestOutput(tb, NewHumanReadableFormatter())
	output.AllowErrorsMatching(errorPattern)
	return GetLogger().WithOutput(output)
}
