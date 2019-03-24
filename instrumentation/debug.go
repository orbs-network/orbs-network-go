// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package instrumentation

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"runtime/debug"
	"runtime/pprof"
)

func DebugPrintCurrentStack(logger log.BasicLogger) {
	bytes := debug.Stack()
	logWriter{logger}.Write(bytes)
}

func DebugPrintGoroutineStacks(logger log.BasicLogger) {
	var buffer bytes.Buffer
	pprof.Lookup("goroutine").WriteTo(&buffer, 1)
	logWriter{logger}.Write(buffer.Bytes())
}

type logWriter struct {
	logger log.BasicLogger
}

func (w logWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}
