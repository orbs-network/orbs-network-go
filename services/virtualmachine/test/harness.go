package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
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

func (h *harness) handleSdkCall(contractName primitives.ContractName, methodName primitives.MethodName, args ...interface{}) ([]*protocol.MethodArgument, error) {
	output, err := h.service.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:      0,
		ContractName:   contractName,
		MethodName:     methodName,
		InputArguments: builders.MethodArguments(args...),
	})
	if err != nil {
		return nil, err
	}
	return output.OutputArguments, nil
}
