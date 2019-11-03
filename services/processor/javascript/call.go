// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package javascript

import (
	"github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/processor"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"plugin"
)

func loadPlugin(path string) (func(handler context.SdkHandler) processor.StatelessProcessor, error) {
	jsPlugin, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	symbol, err := jsPlugin.Lookup("New")
	if err != nil {
		return nil, err
	}

	return symbol.(func(handler context.SdkHandler) processor.StatelessProcessor), nil
}

func (s *service) processMethodCall(executionContextId primitives.ExecutionContextId, code string, methodName primitives.MethodName, args *protocol.ArgumentArray) (contractOutputArgs *protocol.ArgumentArray, contractOutputErr error, err error) {
	w := s.worker(sdk.NewSDK(s.sdkHandler, s.config))
	contractOutArgs, contractOutErr, err := w.ProcessMethodCall(executionContextId, code, methodName, args)
	return contractOutArgs, contractOutErr, err
}
