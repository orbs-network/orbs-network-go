package config

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type validator struct {
	logger log.BasicLogger
}

func NewValidator(logger log.BasicLogger) *validator {
	return &validator{logger: logger}
}

func (v *validator) Validate(cfg NodeConfig) {
	v.requireGT(cfg.BlockSyncNoCommitInterval, cfg.BenchmarkConsensusRetryInterval, "node sync timeout must be greater than benchmark consensus timeout")
	v.requireGT(cfg.BlockSyncNoCommitInterval, cfg.LeanHelixConsensusRoundTimeoutInterval, "node sync timeout must be greater than lean helix round timeout")
}

func (v *validator) requireGT(d1 func() time.Duration, d2 func() time.Duration, msg string) {
	if d1() < d2() {
		panic(fmt.Sprintf("%s; %s=%s is greater than %s=%s", msg, funcName(d1), d1(), funcName(d2), d2()))
	}
}

func funcName(i interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	lastDot := strings.LastIndex(fullName, ".")
	return fullName[lastDot+1:]
}
