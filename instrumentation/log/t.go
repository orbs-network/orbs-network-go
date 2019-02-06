package log

import "time"

// this file is part of test_output.go
// a file with short name (t.go) to make the testLogger prefix less annoying

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
