package test

import (
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
)

type harness struct {
	blockStorage         *services.MockBlockStorage
	stateStorage         *services.MockStateStorage
	processors           map[protocol.ProcessorType]*services.MockProcessor
	crosschainConnectors map[protocol.CrosschainConnectorType]*services.MockCrosschainConnector
	reporting            instrumentation.BasicLogger
	service              services.VirtualMachine
}

func newHarness() *harness {

	log := instrumentation.GetLogger().WithFormatter(instrumentation.NewHumanReadableFormatter())

	blockStorage := &services.MockBlockStorage{}
	stateStorage := &services.MockStateStorage{}

	processors := make(map[protocol.ProcessorType]*services.MockProcessor)
	processors[protocol.PROCESSOR_TYPE_NATIVE] = &services.MockProcessor{}
	processors[protocol.PROCESSOR_TYPE_NATIVE].When("RegisterContractSdkCallHandler", mock.Any).Return().Times(1)

	crosschainConnectors := make(map[protocol.CrosschainConnectorType]*services.MockCrosschainConnector)
	crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM] = &services.MockCrosschainConnector{}

	processorsForService := make(map[protocol.ProcessorType]services.Processor)
	for key, value := range processors {
		processorsForService[key] = value
	}

	crosschainConnectorsForService := make(map[protocol.CrosschainConnectorType]services.CrosschainConnector)
	for key, value := range crosschainConnectors {
		crosschainConnectorsForService[key] = value
	}

	service := virtualmachine.NewVirtualMachine(
		blockStorage,
		stateStorage,
		processorsForService,
		crosschainConnectorsForService,
		log,
	)

	return &harness{
		blockStorage:         blockStorage,
		stateStorage:         stateStorage,
		processors:           processors,
		crosschainConnectors: crosschainConnectors,
		reporting:            log,
		service:              service,
	}
}

func (h *harness) verifyHandlerRegistrations(t *testing.T) {
	for key, processor := range h.processors {
		ok, err := processor.Verify()
		if !ok {
			t.Fatal("Did not register with processor", key.String(), ":", err)
		}
	}
}

func (h *harness) handleSdkCall(contextId primitives.ExecutionContextId, contractName primitives.ContractName, methodName primitives.MethodName, args ...interface{}) ([]*protocol.MethodArgument, error) {
	output, err := h.service.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:      contextId,
		ContractName:   contractName,
		MethodName:     methodName,
		InputArguments: builders.MethodArguments(args...),
	})
	if err != nil {
		return nil, err
	}
	return output.OutputArguments, nil
}

func (h *harness) runLocalMethod() {
	h.service.RunLocalMethod(&services.RunLocalMethodInput{
		BlockHeight: 1,
		Transaction: (&protocol.TransactionBuilder{
			Signer:         nil,
			ContractName:   "ExampleContract",
			MethodName:     "exampleMethod",
			InputArguments: []*protocol.MethodArgumentBuilder{},
		}).Build(),
	})
}

func (h *harness) expectNativeProcessorCalled(f func(primitives.ExecutionContextId)) {
	h.processors[protocol.PROCESSOR_TYPE_NATIVE].When("ProcessCall", mock.Any).Call(func(input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
		f(input.ContextId)
		return &services.ProcessCallOutput{
			OutputArguments: []*protocol.MethodArgument{},
			CallResult:      protocol.EXECUTION_RESULT_SUCCESS,
		}, nil
	}).Times(1)
}

func (h *harness) verifyNativeProcessorCalled(t *testing.T) {
	ok, err := h.processors[protocol.PROCESSOR_TYPE_NATIVE].Verify()
	require.True(t, ok, "did not call processor: %v", err)
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

func (h *harness) expectStateStorageRead(expectedHeight primitives.BlockHeight, expectedKey []byte, returnValue []byte) {
	stateReadMatcher := func(i interface{}) bool {
		input, ok := i.(*services.ReadKeysInput)
		return ok &&
			input.BlockHeight == expectedHeight &&
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
