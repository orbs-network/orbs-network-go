// +build norecover

package supervised

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"os"
	"runtime/debug"
)

func recoverPanics(logger Errorer) {
	if p := recover(); p != nil {
		e := errors.Errorf("goroutine panicked at [%s]: %v", identifyPanic(), p)
		logger.Error("Fatal error", log.Error(e), log.String("stack-trace", string(debug.Stack())))
		os.Exit(42) // because otherwise the system might be a zombie with partial functionality
	}
}
