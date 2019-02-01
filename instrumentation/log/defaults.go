package log

import (
	"testing"
)

func DefaultTestingLogger(tb testing.TB) BasicLogger {
	return GetLogger().WithOutput(NewTestOutput(tb, NewHumanReadableFormatter()))
}
