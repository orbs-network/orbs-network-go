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
	debug.SetTraceback("all")
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
