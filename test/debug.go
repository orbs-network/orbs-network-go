package test

import (
	"os"
	"runtime/pprof"
)

func DebugPrintGoroutineStacks() {
	pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
}
