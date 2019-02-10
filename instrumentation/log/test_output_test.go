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

func (t *fakeTLog) Error(args ...interface{}) {
	t.Called(args...)
}

func (t *fakeTLog) FailNow() {
	t.Called()
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

func TestOutputLogsUnAllowedErrorToTLogAsErrorAndStopsLogging(t *testing.T) {
	m := &fakeTLog{}
	o := NewTestOutput(m, nopFormatter{})
	m.When("Error", "foo").Times(1)
	m.Never("Log", "bar")

	o.Append("error", "foo")
	o.Append("info", "bar")

	_, err := m.Verify()
	require.NoError(t, err)
	require.True(t, o.HasErrors())
}

func TestOutputLogsAllowedErrorToTLogAsInfo(t *testing.T) {
	m := &fakeTLog{}
	o := NewTestOutput(m, nopFormatter{})

	o.AllowErrorsMatching("foo")
	m.When("Log", "foo").Times(1)

	o.Append("error", "foo")

	_, err := m.Verify()
	require.NoError(t, err)
	require.False(t, o.HasErrors())

}
