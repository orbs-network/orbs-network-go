package log

import "time"

func (o *testOutput) Append(level string, message string, fields ...*Field) {
	if o.stopLogging {
		return
	}
	o.tb.Log(o.formatter.FormatRow(time.Now(), level, message, fields...))
	// we use this mechanism to stop logging new log lines after the test failed from a different goroutine
}

func (o *testOutput) StopLogging() {
	o.stopLogging = true
}

// this file is part of test_output.go
// a file with short name (t.go) to make the testLogger prefix less annoying; also we keep o.tb.Log() on a single-digit line for one less char
