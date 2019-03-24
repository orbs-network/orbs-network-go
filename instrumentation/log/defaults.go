// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"testing"
)

func DefaultTestingLogger(tb testing.TB) BasicLogger {
	return GetLogger().WithOutput(NewTestOutput(tb, NewHumanReadableFormatter()))
}

func DefaultTestingLoggerAllowingErrors(tb testing.TB, errorPattern string) BasicLogger {
	output := NewTestOutput(tb, NewHumanReadableFormatter())
	output.AllowErrorsMatching(errorPattern)
	return GetLogger().WithOutput(output)
}
