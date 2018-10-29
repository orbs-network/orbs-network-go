package synchronization

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"runtime"
	"strings"
)

type Errorer interface {
	Error(message string, fields ...*log.Field)
}

func RunSupervised(logger Errorer, f func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				e := errors.Errorf("goroutine panicked at [%s]: %s", identifyPanic(), err)
				logger.Error("recovered panic", log.Error(e))
			}
		}()
		f()
	}()
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

