package supervised

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"runtime"
	"runtime/debug"
	"strings"
)

type Errorer interface {
	Error(message string, fields ...*log.Field)
}

// Runs f() in a goroutine; if it panics, logs the error and stack trace to the specified Errorer
func ShortLived(logger Errorer, f func()) {
	go func() {
		defer recoverPanics(logger)
		f()
	}()
}

// Runs f() in a goroutine; if it panics, logs the error and stack trace to the specified Errorer; if the provided Context isn't closed, re-runs f()
func LongLived(ctx context.Context, logger Errorer, f func()) {
	go func() {
		defer recoverPanics(logger)
		for {
			f()
			if ctx.Err() != nil { // this returns non-nil when context has been closed via cancellation or timeout or whatever
				return
			}
			// repeat
			//TODO count restarts, fail if too many restarts, etc
		}
	}()
}

func recoverPanics(logger Errorer) {
	if p := recover(); p != nil {
		e := errors.Errorf("goroutine panicked at [%s]: %v", identifyPanic(), p)
		logger.Error("recovered panic", log.Error(e), log.String("stack-trace", string(debug.Stack())))
	}
}

func identifyPanic() string {
	var name, file string
	var line int
	var pc [16]uintptr

	n := runtime.Callers(3, pc[:])
	for _, pc := range pc[:n] {
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		file, line = fn.FileLine(pc)
		name = fn.Name()
		if !strings.HasPrefix(name, "runtime.") {
			break
		}
	}

	switch {
	case name != "":
		return fmt.Sprintf("%v:%v", name, line)
	case file != "":
		return fmt.Sprintf("%v:%v", file, line)
	}

	return fmt.Sprintf("pc:%x", pc)
}
