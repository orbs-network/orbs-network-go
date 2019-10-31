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
