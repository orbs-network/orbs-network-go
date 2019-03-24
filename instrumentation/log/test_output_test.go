// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"github.com/orbs-network/go-mock"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

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
	m.When("Error", TEST_FAILED_ERROR).Times(1)
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

func TestOutputStopsRecordingErrorsAfterTestTerminated(t *testing.T) {
	m := &fakeTLog{}
	o := NewTestOutput(m, nopFormatter{})

	m.When("Error", "foo").Times(0)

	o.TestTerminated()
	o.Append("error", "foo")

	_, err := m.Verify()
	require.NoError(t, err)
}

func TestOutputRecoversFromTestRunnerPanicsDuringRecordError(t *testing.T) {
	m := &fakeTLog{}
	o := NewTestOutput(m, nopFormatter{})

	m.When("Error", "foo").Call(func(string) {
		panic("test runner panic")
	})

	require.NotPanics(t, func() {
		o.Append("error", "foo")
	})
}

type fakeTLog struct {
	mock.Mock
}

func (t *fakeTLog) Error(args ...interface{}) {
	t.Called(args...)
}

func (t *fakeTLog) Fatal(args ...interface{}) {
	t.Called(args...)
}

func (t *fakeTLog) Log(args ...interface{}) {
	t.Called(args...)
}

func (t *fakeTLog) Name() string {
	return "FakeTestName"
}

type nopFormatter struct {
}

func (nopFormatter) FormatRow(timestamp time.Time, level string, message string, params ...*Field) (formattedRow string) {
	return message
}
