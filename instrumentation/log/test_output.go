package log

import (
	"testing"
	"time"
)

func NewTestOutput(tb testing.TB, formatter LogFormatter) *testOutput {
	return &testOutput{tb: tb, formatter: formatter}
}

type testOutput struct {
	formatter   LogFormatter
	tb          testing.TB
	stopLogging bool
}

func (o *testOutput) Append(level string, message string, fields ...*Field) {
	// we use this mechanism to stop logging new log lines after the test failed from a different goroutine
	if o.stopLogging {
		return
	}

	logLine := o.formatter.FormatRow(time.Now(), level, message, fields...)
	o.tb.Log(logLine)
}

func (o *testOutput) StopLogging() {
	o.stopLogging = true
}
