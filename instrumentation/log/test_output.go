package log

import (
	"testing"
)

func NewTestOutput(tb testing.TB, formatter LogFormatter) *testOutput {
	return &testOutput{tb: tb, formatter: formatter}
}

type testOutput struct {
	formatter   LogFormatter
	tb          testing.TB
	stopLogging bool
}

// func (o *testOutput) Append(level string, message string, fields ...*Field) moved to file t.go
