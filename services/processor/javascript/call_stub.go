// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build !jsprocessor

package javascript

import (
	"github.com/netoneko/orbs-network-javascript-plugin/worker"
	"github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"plugin"
)

func loadPlugin() (*func(handler context.SdkHandler) worker.Worker, error) {
	jsPlugin, err := plugin.Open("/Users/kirill/gopath/src/github.com/netoneko/orbs-network-javascript-plugin/test/main.bin")
	if err != nil {
		return nil, err
	}

	symbol, err := jsPlugin.Lookup("New")
	if err != nil {
		return nil, err
	}

	return symbol.(*func(handler context.SdkHandler) worker.Worker), nil
}

func (s *service) processMethodCall(executionContextId primitives.ExecutionContextId, code string, methodName primitives.MethodName, args *protocol.ArgumentArray) (contractOutputArgs *protocol.ArgumentArray, contractOutputErr error, err error) {
	//panic("not implemented")

	w := (*s.worker)(s)
	contractOutArgs, contractOutErr, err := w.ProcessMethodCall([]byte(executionContextId), code, string(methodName), args.Raw())
	return protocol.ArgumentArrayReader(contractOutArgs), contractOutErr, err
}
