package test

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

func WithContext(f func(ctx context.Context)) {
	fmt.Println("= RUNNING TEST  ", getCallerFuncName(), " (test.WithContext hack)") // we added this to workaround https://github.com/orbs-network/orbs-network-go/issues/377
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f(ctx)
}

func getCallerFuncName() string {
	pc, _, _, _ := runtime.Caller(2)
	packageAndFuncName := runtime.FuncForPC(pc).Name()
	parts := strings.Split(packageAndFuncName, ".")
	return parts[len(parts)-1]
}
