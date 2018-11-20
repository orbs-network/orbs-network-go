//+build nonativecompiler

package adapter

import "github.com/orbs-network/orbs-network-go/instrumentation/log"

func NewNativeCompiler(config Config, logger log.BasicLogger) Compiler {
	return nil
}