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

type failure struct {
	stackTrace string
	e          error
}

// Runs f() in a goroutine; if it panics, logs the error and stack trace to the specified Errorer
func ShortLived(logger Errorer, f func()) {
	go func() {
		defer func() {
			if p := recover(); p != nil {
				e := errors.Errorf("goroutine panicked at [%s]: %v", identifyPanic(), p)
				logger.Error("recovered panic", log.Error(e), log.String("stack-trace", string(debug.Stack())))
			}
		}()
		f()
	}()
}

// Runs f() in a goroutine; if it panics, logs the error and stack trace to the specified Errorer; if the provided Context isn't closed, re-runs f()
func LongLived(ctx context.Context, logger Errorer, f func()) {
	failed := make(chan *failure)

	run := func() {
		defer func() {
			if p := recover(); p != nil {
				e := errors.Errorf("goroutine panicked at [%s]: %v", identifyPanic(), p)
				failed <- &failure{e: e, stackTrace: string(debug.Stack())}
			}
		}()

		f()
	}

	supervise := func() {
		for {
			select {
			case <-ctx.Done():
				return
			case failure := <-failed:
				//TODO count restarts, fail if too many restarts, etc

				if ctx.Err() == nil {
					logger.Error("recovered panic", log.Error(failure.e), log.String("stack-trace", failure.stackTrace))
					go run()
				}

			}
		}
	}

	go supervise()
	go run()
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
