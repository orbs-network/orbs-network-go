package supervized

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"runtime"
	"strings"
)

type Errorer interface {
	Error(message string, fields ...*log.Field)
}

func OneOff(logger Errorer, f func()) {
	go func() {
		defer recoverPanics(logger)
		f()
	}()
}

func LongLiving(ctx context.Context, logger Errorer, f func()) {
	defer recoverPanics(logger)
	go func() {
		for {
			f()
			select {
			case <-ctx.Done():
				return
			default:
				// repeat
				//TODO count restarts, fail if too many restarts, etc
			}
		}
	}()
}

func recoverPanics(logger Errorer) {
	if err := recover(); err != nil {
		e := errors.Errorf("goroutine panicked at [%s]: %s", identifyPanic(), err)
		logger.Error("recovered panic", log.Error(e))
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
