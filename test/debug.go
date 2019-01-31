package test

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"runtime/pprof"
)

func DebugPrintGoroutineStacks(logger log.BasicLogger) {
	pprof.Lookup("goroutine").WriteTo(logWriter{logger}, 1)
}

type logWriter struct {
	logger log.BasicLogger
}

func (w logWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}
