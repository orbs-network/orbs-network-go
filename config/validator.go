package config

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"
)

func Validate(cfg NodeConfig) {
	requireGT(cfg.BlockSyncNoCommitInterval, cfg.BenchmarkConsensusRetryInterval, "node sync timeout must be less than benchmark consensus timeout")
	requireGT(cfg.BlockSyncNoCommitInterval, cfg.LeanHelixConsensusRoundTimeoutInterval, "node sync timeout must be less than lean helix round timeout")
}

func requireGT(d1 func() time.Duration, d2 func() time.Duration, msg string) {
	if d1() < d2() {
		panic(fmt.Sprintf("%s; %s=%s is greater than %s=%s", msg, funcName(d1), d1(), funcName(d2), d2()))
	}
}

func funcName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	lastDot := strings.LastIndex(fullName, ".")
	return fullName[lastDot+1:]
}
