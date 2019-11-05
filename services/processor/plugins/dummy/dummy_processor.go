// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build javascript

package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/processor/arguments"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type dummyProcessor struct {
	handler context.SdkHandler
}

func (d *dummyProcessor) ProcessMethodCall(executionContextId primitives.ExecutionContextId, code string, methodName primitives.MethodName, args *protocol.ArgumentArray) (contractOutputArgs *protocol.ArgumentArray, contractOutputErr error, err error) {
	println("called dummy processor!")
	output := arguments.ArgsToArgumentArray(contracts.MOCK_COUNTER_CONTRACT_START_FROM)
	return output, nil, nil
}
