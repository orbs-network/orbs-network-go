package log

import (
	"fmt"
	"io"
	"time"
)

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
