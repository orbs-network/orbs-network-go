// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package _supervised_in_test

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	logLine "github.com/orbs-network/orbs-network-go/synchronization/supervised/test"
	"testing"
	"time"
)

func Test_Panics(t *testing.T) {
	t.Log(logLine.BeforeLoggerCreated)
	testLogger := log.DefaultTestingLogger(t)
	testLogger.Info(logLine.LoggedWithLogger)

	t.Log(logLine.BeforeCallPanic)
	theFunctionThrowingThePanic()
	t.Log(logLine.AfterCallPanic)

	testLogger.Info(logLine.MustNotShow)
	t.Log(logLine.MustNotShow)
}

func Test_LogsError(t *testing.T) {
	t.Log(logLine.BeforeLoggerCreated)
	testLogger := log.DefaultTestingLogger(t)
	testLogger.Info(logLine.LoggedWithLogger)

	t.Log(logLine.BeforeLoggerError)
	testLogger.Error(logLine.ErrorWithLogger)
	t.Log(logLine.AfterLoggerError)

	testLogger.Info(logLine.MustNotShow)
	t.Log(logLine.MustShow)
}

func TestGoOnce_Panics(t *testing.T) {
	t.Log(logLine.BeforeLoggerCreated)
	testLogger := log.DefaultTestingLogger(t)
	testLogger.Info(logLine.LoggedWithLogger)

	supervised.GoOnce(testLogger, func() {
		t.Log(logLine.BeforeCallPanic)
		theFunctionThrowingThePanic()
		t.Log(logLine.AfterCallPanic)

		testLogger.Info(logLine.MustNotShow)
		t.Log(logLine.MustNotShow)
	})

	// Test lingers for the exception in goroutine to arrive
	time.Sleep(100 * time.Millisecond)

	testLogger.Info(logLine.MustNotShow)
	t.Log(logLine.MustShow)
}

func TestGoOnce_LogsError(t *testing.T) {
	t.Log(logLine.BeforeLoggerCreated)
	testLogger := log.DefaultTestingLogger(t)
	testLogger.Info(logLine.LoggedWithLogger)

	supervised.GoOnce(testLogger, func() {
		t.Log(logLine.BeforeLoggerError)
		testLogger.Error(logLine.ErrorWithLogger)
		t.Log(logLine.AfterLoggerError)

		testLogger.Info(logLine.MustNotShow)
		t.Log(logLine.MustShow)
	})

	// Test lingers for the exception in goroutine to arrive
	time.Sleep(100 * time.Millisecond)

	testLogger.Info(logLine.MustNotShow)
	t.Log(logLine.MustShow)
}

func TestTRun_Panics(t *testing.T) {
	t.Run("SubTest", func(t *testing.T) {
		t.Log(logLine.BeforeLoggerCreated)
		subTestLogger := log.DefaultTestingLogger(t)
		subTestLogger.Info(logLine.LoggedWithLogger)
		supervised.Recover(subTestLogger, func() { // t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside

			t.Log(logLine.BeforeCallPanic)
			theFunctionThrowingThePanic()
			t.Log(logLine.AfterCallPanic)

			subTestLogger.Info(logLine.MustNotShow)
			t.Log(logLine.MustNotShow)
		})
	})
}

func TestTRun_LogsError(t *testing.T) {
	t.Run("SubTest", func(t *testing.T) {
		t.Log(logLine.BeforeLoggerCreated)
		subTestLogger := log.DefaultTestingLogger(t)
		subTestLogger.Info(logLine.LoggedWithLogger)
		supervised.Recover(subTestLogger, func() { // t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside

			t.Log(logLine.BeforeLoggerError)
			subTestLogger.Error(logLine.ErrorWithLogger)
			t.Log(logLine.AfterLoggerError)

			subTestLogger.Info(logLine.MustNotShow)
			t.Log(logLine.MustShow)
		})
	})
}

func TestTRun_GoOnce_Panics(t *testing.T) {
	t.Run("SubTest", func(t *testing.T) {
		t.Log(logLine.BeforeLoggerCreated)
		subTestLogger := log.DefaultTestingLogger(t)
		subTestLogger.Info(logLine.LoggedWithLogger)
		supervised.Recover(subTestLogger, func() { // t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside

			supervised.GoOnce(subTestLogger, func() {
				t.Log(logLine.BeforeCallPanic)
				theFunctionThrowingThePanic()
				t.Log(logLine.AfterCallPanic)

				subTestLogger.Info(logLine.MustNotShow)
				t.Log(logLine.MustNotShow)
			})

			// SubTest lingers for the exception in goroutine to arrive
			time.Sleep(100 * time.Millisecond)
			subTestLogger.Info(logLine.MustNotShow)
			t.Log(logLine.MustShow)
		})
	})
}

func TestTRun_GoOnce_LogsError(t *testing.T) {
	t.Run("SubTest", func(t *testing.T) {
		t.Log(logLine.BeforeLoggerCreated)
		subTestLogger := log.DefaultTestingLogger(t)
		subTestLogger.Info(logLine.LoggedWithLogger)
		supervised.Recover(subTestLogger, func() { // t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside

			supervised.GoOnce(subTestLogger, func() {
				t.Log(logLine.BeforeLoggerError)
				subTestLogger.Error(logLine.ErrorWithLogger)
				t.Log(logLine.AfterLoggerError)

				subTestLogger.Info(logLine.MustNotShow)
				t.Log(logLine.MustShow)
			})

			// SubTest lingers for the exception in goroutine to arrive
			time.Sleep(100 * time.Millisecond)
			subTestLogger.Info(logLine.MustNotShow)
			t.Log(logLine.MustShow)
		})
	})
}

func TestTRun_GoOnce_PanicsAfterSubTestPasses(t *testing.T) {
	subTestPassedChannel := make(chan bool)

	t.Run("SubTest", func(t *testing.T) {
		t.Log(logLine.BeforeLoggerCreated)
		testOutput := log.NewTestOutput(t, log.NewHumanReadableFormatter())
		subTestLogger := log.GetLogger().WithOutput(testOutput)
		subTestLogger.Info(logLine.LoggedWithLogger)
		supervised.Recover(subTestLogger, func() { // t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside
			defer testOutput.TestTerminated() // this is required to prevent crash

			supervised.GoOnce(subTestLogger, func() {
				<-subTestPassedChannel
				// still in goroutine after the SubTest passes

				t.Log(logLine.BeforeCallPanic)
				theFunctionThrowingThePanic()
				t.Log(logLine.AfterCallPanic)

				subTestLogger.Info(logLine.MustNotShow)
				t.Log(logLine.MustNotShow)
			})

			// SubTest is now passing successfully since it returns without any issues
		})
	})

	// ParentTest is now post a successfully passing SubTest
	subTestPassedChannel <- true
	// ParentTest lingers for the exception in goroutine to arrive
	time.Sleep(100 * time.Millisecond)
}

func TestTRun_GoOnce_LogsErrorAfterSubTestPasses(t *testing.T) {
	subTestPassedChannel := make(chan bool)

	t.Run("SubTest", func(t *testing.T) {
		t.Log(logLine.BeforeLoggerCreated)
		testOutput := log.NewTestOutput(t, log.NewHumanReadableFormatter())
		subTestLogger := log.GetLogger().WithOutput(testOutput)
		subTestLogger.Info(logLine.LoggedWithLogger)
		supervised.Recover(subTestLogger, func() { // t.Run is dangerous because it creates an unsupervised goroutine, we must use Recover inside
			defer testOutput.TestTerminated() // this is required to prevent crash

			supervised.GoOnce(subTestLogger, func() {
				<-subTestPassedChannel
				// still in goroutine after the SubTest passes

				t.Log(logLine.BeforeLoggerError)
				subTestLogger.Error(logLine.ErrorWithLogger)
				t.Log(logLine.AfterLoggerError)

				subTestLogger.Info(logLine.MustNotShow)
				t.Log(logLine.MustShow)
			})

			// SubTest is now passing successfully since it returns without any issues
		})
	})

	// ParentTest is now post a successfully passing SubTest
	subTestPassedChannel <- true
	// ParentTest lingers for the exception in goroutine to arrive
	time.Sleep(100 * time.Millisecond)
}

func theFunctionThrowingThePanic() {
	panic("oh no")
}
