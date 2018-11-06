package log

import (
	"fmt"
	"io"
)

type Output interface {
	Append(level string, message string, fields ...*Field)
}

type basicOutput struct {
	formatter LogFormatter
	writer    io.Writer
}

func (out *basicOutput) Append(level string, message string, fields ...*Field) {
	logLine := out.formatter.FormatRow(level, message, fields...)
	fmt.Fprintln(out.writer, logLine)
}

func NewFormattingOutput(writer io.Writer, formatter LogFormatter) Output {
	return &basicOutput{formatter, writer}
}


