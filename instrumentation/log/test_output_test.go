package log

import (
	"github.com/orbs-network/go-mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type fakeTLog struct {
	mock.Mock
}

func (t *fakeTLog) FailNow() {
	t.Called("FailNow")
}

func (t *fakeTLog) Log(args ...interface{}) {
	t.Called(args...)
}

type nopFormatter struct {
}

func (nopFormatter) FormatRow(timestamp time.Time, level string, message string, params ...*Field) (formattedRow string) {
	return message
}

func TestTestOutputLogsToTLog(t *testing.T) {
	m := &fakeTLog{}
	o := NewTestOutput(m, nopFormatter{})
	m.When("Log", "foo").Times(1)

	o.Append("info", "foo")

	_, err := m.Verify()
	require.NoError(t, err)
}

func TestTestOutputDoesNotLogToTLogAfterStopLoggingWasCalled(t *testing.T) {
	m := &fakeTLog{}
	o := NewTestOutput(m, nopFormatter{})
	m.When("Log", "foo").Times(1)

	m.Never("Log", "bar")

	o.Append("info", "foo")
	o.StopLogging()
	o.Append("info", "bar")

	_, err := m.Verify()
	require.NoError(t, err)

}
