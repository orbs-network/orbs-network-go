package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

type harness struct {
	sdkCallHandler *handlers.MockContractSdkCallHandler
	service        services.Processor
}

func newHarness() *harness {
	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	sdkCallHandler := &handlers.MockContractSdkCallHandler{}

	service := native.NewNativeProcessor(log)
	service.RegisterContractSdkCallHandler(sdkCallHandler)

	return &harness{
		sdkCallHandler: sdkCallHandler,
		service:        service,
	}
}

func (h *harness) expectSdkCallMadeWithStateRead(returnValue []byte) {
	stateReadCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.ContractName == native.SDK_STATE_CONTRACT_NAME &&
			input.MethodName == "read"
	}

	readReturn := &handlers.HandleSdkCallOutput{
		OutputArguments: builders.MethodArguments(returnValue),
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.AnyIf("Contract equals Sdk.State and method equals read", stateReadCallMatcher)).Return(readReturn, nil).Times(1)
}

func (h *harness) expectSdkCallMadeWithStateWrite() {
	stateWriteCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.ContractName == native.SDK_STATE_CONTRACT_NAME &&
			input.MethodName == "write"
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.AnyIf("Contract equals Sdk.State and method equals write", stateWriteCallMatcher)).Return(nil, nil).Times(1)
}

func (h *harness) verifySdkCallMade(t *testing.T) {
	ok, err := h.sdkCallHandler.Verify()
	require.True(t, ok, "sdkCallHandler should run as expected", err)
}
