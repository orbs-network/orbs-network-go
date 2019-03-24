// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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

type ContextEndedChan chan struct{}

// Runs f() in a new goroutine; if it panics, logs the error and stack trace to the specified Errorer
func GoOnce(errorer Errorer, f func()) {
	go func() {
		tryOnce(errorer, f)
	}()
}

// Runs f() in a new goroutine; if it panics, logs the error and stack trace to the specified Errorer
// If the provided Context isn't closed, re-runs f()
// Returns a channel that is closed when the goroutine has quit due to context ending
func GoForever(ctx context.Context, logger Errorer, f func()) ContextEndedChan {
	c := make(ContextEndedChan)
	go func() {
		defer close(c)

		for {
			tryOnce(logger, f)
			//TODO(v1) report number of restarts to metrics
			if ctx.Err() != nil { // this returns non-nil when context has been closed via cancellation or timeout or whatever
				return
			}
		}
	}()
	return c
}

// Runs f() on the original goroutine; if it panics, logs the error and stack trace to the specified Errorer
// Very similar to GoOnce except doesn't start a new goroutine
func Recover(errorer Errorer, f func()) {
	tryOnce(errorer, f)
}

// this function is needed so that we don't return out of the goroutine when it panics
func tryOnce(errorer Errorer, f func()) {
	defer recoverPanics(errorer)
	f()
}

func recoverPanics(logger Errorer) {
	if p := recover(); p != nil {
		e := errors.Errorf("\npanic: %v\n\ngoroutine panicked at:\n%s\n\n", p, identifyPanic())
		logger.Error("recovered panic", log.Error(e), log.String("panic", "true"), log.String("stack-trace", string(debug.Stack())))
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
