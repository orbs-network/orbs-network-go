package log

import (
	"fmt"
	"io"
	"testing"
	"time"
)

type Output interface {
	Append(level string, message string, fields ...*Field)
}

type basicOutput struct {
	formatter LogFormatter
	writer    io.Writer
}

func (out *basicOutput) Append(level string, message string, fields ...*Field) {
	logLine := out.formatter.FormatRow(time.Now(), level, message, fields...)
	fmt.Fprintln(out.writer, logLine)
}

func NewFormattingOutput(writer io.Writer, formatter LogFormatter) Output {
	return &basicOutput{formatter, writer}
}

func NewTestOutput(tb testing.TB, formatter LogFormatter) *testOutput {
	return &testOutput{tb: tb, formatter: formatter}
}

type testOutput struct {
	formatter LogFormatter
	tb        testing.TB
}
