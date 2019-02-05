package log

import "time"

func (o *testOutput) Append(level string, message string, fields ...*Field) {
	logLine := o.formatter.FormatRow(time.Now(), level, message, fields...)
	o.tb.Log(logLine)
}

// Keep the Log() line in Append() under line 10, this saves a char when printing that EVERY SINGLE LINE
