package log

import (
	"sync"
)

type ErrorRecordingLogger struct {
	nested        BasicLogger
	allowedErrors []string

	errorRecorder *errorRecorder
}

type errorRecorder struct {
	sync.Mutex
	unexpectedErrors []string
}

func NewErrorRecordingLogger(wrapped BasicLogger, allowedErrors []string) *ErrorRecordingLogger {
	return &ErrorRecordingLogger{nested: wrapped, allowedErrors: allowedErrors, errorRecorder: &errorRecorder{}}
}

func (l *ErrorRecordingLogger) Log(level string, message string, params ...*Field) {
	l.nested.Log(level, message, params...)

}

func (l *ErrorRecordingLogger) LogFailedExpectation(message string, expected *Field, actual *Field, params ...*Field) {
	l.nested.LogFailedExpectation(message, expected, actual, params...)
}

func (l *ErrorRecordingLogger) Info(message string, params ...*Field) {
	l.nested.Info(message, params...)
}

func (l *ErrorRecordingLogger) Error(message string, params ...*Field) {
	l.nested.Error(message, params...)

	for _, allowedMessage := range l.allowedErrors {
		if allowedMessage == message {
			return
		}
	}

	l.recordUnexpectedError(message, params)
}

func (l *ErrorRecordingLogger) Metric(params ...*Field) {
	l.nested.Metric(params...)
}

func (l *ErrorRecordingLogger) WithTags(params ...*Field) BasicLogger {
	return &ErrorRecordingLogger{
		nested: l.nested.WithTags(),
		allowedErrors: l.allowedErrors,
		errorRecorder: l.errorRecorder,
	}

	return NewErrorRecordingLogger(l.nested.WithTags(), l.allowedErrors)
}

func (l *ErrorRecordingLogger) Tags() []*Field {
	return l.nested.Tags()
}

func (l *ErrorRecordingLogger) WithOutput(writer ...Output) BasicLogger {
	return l.nested.WithOutput(writer...)
}

func (l *ErrorRecordingLogger) WithFilters(filter ...Filter) BasicLogger {
	return l.nested.WithFilters(filter...)
}

func (l *ErrorRecordingLogger) recordUnexpectedError(message string, fields []*Field) {
	l.errorRecorder.Lock()
	defer l.errorRecorder.Unlock()
	l.errorRecorder.unexpectedErrors = append(l.errorRecorder.unexpectedErrors, message)

}

func (l *ErrorRecordingLogger) GetUnexpectedErrors() []string {
	return l.errorRecorder.unexpectedErrors
}

func (l *ErrorRecordingLogger) HasErrors() bool {
	l.errorRecorder.Lock()
	defer l.errorRecorder.Unlock()
	return len(l.errorRecorder.unexpectedErrors) > 0
}


