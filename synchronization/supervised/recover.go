// +build !norecover

package supervised

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"runtime/debug"
)

func recoverPanics(logger Errorer) {
	if p := recover(); p != nil {
		e := errors.Errorf("goroutine panicked at [%s]: %v", identifyPanic(), p)
		logger.Error("recovered panic", log.Error(e), log.String("stack-trace", string(debug.Stack())))
	}
}
