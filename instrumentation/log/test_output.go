package log

type TLog interface {
	FailNow()
	Log(args ...interface{})
}

func NewTestOutput(tb TLog, formatter LogFormatter) *testOutput {
	return &testOutput{tb: tb, formatter: formatter}
}

type testOutput struct {
	formatter   LogFormatter
	tb          TLog
	stopLogging bool
}

// func (o *testOutput) Append(level string, message string, fields ...*Field) moved to file t.go
