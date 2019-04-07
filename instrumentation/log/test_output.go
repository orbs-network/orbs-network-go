// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"fmt"
	"regexp"
)

const TEST_FAILED_ERROR = "Test failed due to unexpected errors being logged. If the error above is expected, please add it to the list of allowed errors by invoking TestOutput.AllowErrorsMatching"
const POST_TERMINATED_ERROR = "*** Logged error after TestOutput.TestTerminated:"
const TEST_RUNNER_PANIC_ERROR = "*** Test runner panic while trying to fail test (try using TestOutput.TestTerminated):"

type TLog interface {
	Fatal(args ...interface{})
	Log(args ...interface{})
	Error(args ...interface{})
	Name() string
}

func NewTestOutput(tb TLog, formatter LogFormatter) *TestOutput {
	return &TestOutput{tb: tb, formatter: formatter}
}

type TestOutput struct {
	formatter            LogFormatter
	tb                   TLog
	stopLogging          bool
	allowedErrors        []string
	allowedErrorPatterns []*regexp.Regexp
	hasErrors            bool
	testTerminated       bool
}

func (o *TestOutput) allowed(message string, fields []*Field) bool {
	for _, allowedPattern := range o.allowedErrorPatterns {
		if allowedPattern.MatchString(message) {
			return true
		}
		for _, f := range fields {
			if f.Key == "error" {
				if allowedPattern.MatchString(f.String()) {
					return true
				}
			}
		}
	}

	return false
}

func (o *TestOutput) AllowErrorsMatching(pattern string) {
	compiledPattern, _ := regexp.Compile(pattern)
	o.allowedErrors = append(o.allowedErrors, pattern)
	o.allowedErrorPatterns = append(o.allowedErrorPatterns, compiledPattern)
}

func (o *TestOutput) HasErrors() bool {
	return o.hasErrors
}

// the golang test runner throws a severe panic if trying to fail a test after it already passed
// this happens for example on t.Run where a goroutine logs an Error (which fails the test) after t.Run passed
// the solution is to add "defer testOutput.TestTerminated()" to execute as the t.Run body is returning
func (o *TestOutput) TestTerminated() {
	o.testTerminated = true
}

func (o *TestOutput) recordError(line string) {
	defer func() {
		if p := recover(); p != nil {
			// a known panic is when we try to fail from a goroutine a test that already passed
			fmt.Println(TEST_RUNNER_PANIC_ERROR, o.tb.Name(), ":", p, ":", line)
		}
	}()

	o.hasErrors = true
	if !o.testTerminated {

		o.tb.Error(line)
		o.tb.Error(TEST_FAILED_ERROR)

	} else {

		// must use print because after test is terminated its t.Log does not output anything
		fmt.Println(POST_TERMINATED_ERROR, o.tb.Name(), ":", line)

	}
}

// func (o *TestOutput) Append(level string, message string, fields ...*Field) moved to file t.go
