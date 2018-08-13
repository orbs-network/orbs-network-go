package test

import (
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func (h *harness) verifyHandlerRegistrations(t *testing.T) {
	for key, processor := range h.processors {
		ok, err := processor.Verify()
		if !ok {
			t.Fatal("Did not register with processor", key.String(), ":", err)
		}
	}
}

func (h *harness) expectNativeContractMethodCalled(expectedContractName primitives.ContractName, expectedMethodName primitives.MethodName, contractFunction func(primitives.ExecutionContextId) (protocol.ExecutionResult, error)) {
	contractMethodMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ProcessCallInput)
		return ok &&
			input.ContractName == expectedContractName &&
			input.MethodName == expectedMethodName
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("ProcessCall", mock.AnyIf(fmt.Sprintf("Contract equals %s and Method %s", expectedContractName, expectedMethodName), contractMethodMatcher)).Call(func(input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
		callResult, err := contractFunction(input.ContextId)
		return &services.ProcessCallOutput{
			OutputArguments: []*protocol.MethodArgument{},
			CallResult:      callResult,
		}, err
	}).Times(1)
}

func (h *harness) verifyNativeContractMethodCalled(t *testing.T) {
	ok, err := h.processors[protocol.PROCESSOR_TYPE_NATIVE].Verify()
	require.True(t, ok, "did not call processor: %v", err)
}

func (h *harness) expectSystemContractCalled(expectedContractName primitives.ContractName, expectedMethodName primitives.MethodName, returnError error) {
	contractMethodMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ProcessCallInput)
		return ok &&
			input.ContractName == expectedContractName &&
			input.MethodName == expectedMethodName
	}

	outputToReturn := &services.ProcessCallOutput{
		OutputArguments: []*protocol.MethodArgument{},
		CallResult:      protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT,
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("ProcessCall", mock.AnyIf(fmt.Sprintf("Contract equals %s and Method %s", expectedContractName, expectedMethodName), contractMethodMatcher)).Return(outputToReturn, returnError).Times(1)
}

func (h *harness) verifySystemContractCalled(t *testing.T) {
	ok, err := h.processors[protocol.PROCESSOR_TYPE_NATIVE].Verify()
	require.True(t, ok, "did not call processor for system contract: %v", err)
}

func (h *harness) expectNativeContractInfoRequested(expectedContractName primitives.ContractName, returnError error) {
	contractMatcher := func(i interface{}) bool {
		input, ok := i.(*services.GetContractInfoInput)
		return ok &&
			input.ContractName == expectedContractName
	}

	outputToReturn := &services.GetContractInfoOutput{
		PermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
	}

	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("GetContractInfo", mock.AnyIf(fmt.Sprintf("Contract equals %s", expectedContractName), contractMatcher)).Return(outputToReturn, returnError).Times(1)
}

func (h *harness) verifyNativeContractInfoRequested(t *testing.T) {
	ok, err := h.processors[protocol.PROCESSOR_TYPE_NATIVE].Verify()
	require.True(t, ok, "did not request contract info: %v", err)
}

func (h *harness) expectStateStorageBlockHeightRequested(returnValue primitives.BlockHeight) {
	outputToReturn := &services.GetStateStorageBlockHeightOutput{
		LastCommittedBlockHeight:    returnValue,
		LastCommittedBlockTimestamp: 1234,
	}

	h.stateStorage.When("GetStateStorageBlockHeight", mock.Any).Return(outputToReturn, nil).Times(1)
}

func (h *harness) verifyStateStorageBlockHeightRequested(t *testing.T) {
	ok, err := h.stateStorage.Verify()
	require.True(t, ok, "did not read from state storage: %v", err)
}

func (h *harness) expectStateStorageRead(expectedHeight primitives.BlockHeight, expectedContractName primitives.ContractName, expectedKey []byte, returnValue []byte) {
	stateReadMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ReadKeysInput)
		return ok &&
			input.BlockHeight == expectedHeight &&
			input.ContractName == expectedContractName &&
			len(input.Keys) == 1 &&
			input.Keys[0].Equal(expectedKey)
	}

	outputToReturn := &services.ReadKeysOutput{
		StateRecords: []*protocol.StateRecord{(&protocol.StateRecordBuilder{
			Key:   expectedKey,
			Value: returnValue,
		}).Build()},
	}

	h.stateStorage.When("ReadKeys", mock.AnyIf(fmt.Sprintf("ReadKeys height equals %s and key equals %x", expectedHeight, expectedKey), stateReadMatcher)).Return(outputToReturn, nil).Times(1)
}

func (h *harness) verifyStateStorageRead(t *testing.T) {
	ok, err := h.stateStorage.Verify()
	require.True(t, ok, "did not read from state storage: %v", err)
}

func (h *harness) expectStateStorageNotRead() {
	h.stateStorage.When("ReadKeys", mock.Any).Return(&services.ReadKeysOutput{}, nil).Times(0)
}
