package log

import "io"

type Output interface {
	Output() io.Writer
	Formatter() LogFormatter
	WithFormatter(f LogFormatter) Output
}

type basicOutput struct {
	formatter LogFormatter
	output    io.Writer
}

func (out *basicOutput) Formatter() LogFormatter {
	return out.formatter
}

func (out *basicOutput) Output() io.Writer {
	return out.output
}

func (out *basicOutput) WithFormatter(f LogFormatter) Output {
	out.formatter = f
	return out
}

func NewOutput(writer io.Writer) Output {
	return &basicOutput{NewJsonFormatter(), writer}
}
