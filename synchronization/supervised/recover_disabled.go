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
		stack := string(debug.Stack())
		logger.Error("Fatal error", log.Error(e), log.String("stack-trace", stack))
		println("Exited brutally due to panicking goroutine: ", e)
		println(stack)
		os.Exit(42) // because otherwise the system might be a zombie with partial functionality
	}
}
