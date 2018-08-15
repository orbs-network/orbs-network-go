package instrumentation

import "io"

type BasicOutput interface {
	Output() io.Writer
	Formatter() LogFormatter
	WithFormatter(f LogFormatter) BasicOutput
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

func (out *basicOutput) WithFormatter(f LogFormatter) BasicOutput {
	out.formatter = f
	return out
}

func Output(writer io.Writer) BasicOutput {
	return &basicOutput{NewJsonFormatter(), writer}
}
