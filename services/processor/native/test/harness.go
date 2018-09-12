package test

import (
	"bytes"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
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
			input.OperationName == native.SDK_OPERATION_NAME_STATE &&
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
			input.OperationName == native.SDK_OPERATION_NAME_STATE &&
			input.MethodName == "write"
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.AnyIf("Contract equals Sdk.State and method equals write", stateWriteCallMatcher)).Return(nil, nil).Times(1)
}

func (h *harness) expectSdkCallMadeWithServiceCallMethod(expectedContractName primitives.ContractName, expectedMethodName primitives.MethodName, expectedArgArray *protocol.MethodArgumentArray, returnError error) {
	serviceCallMethodCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == native.SDK_OPERATION_NAME_SERVICE &&
			input.MethodName == "callMethod" &&
			len(input.InputArguments) == 3 &&
			input.InputArguments[0].StringValue() == string(expectedContractName) &&
			input.InputArguments[1].StringValue() == string(expectedMethodName) &&
			bytes.Equal(input.InputArguments[2].BytesValue(), expectedArgArray.Raw())
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.AnyIf("Contract equals Sdk.Service, method equals callMethod and 3 args match", serviceCallMethodCallMatcher)).Return(nil, returnError).Times(1)
}

func (h *harness) verifySdkCallMade(t *testing.T) {
	_, err := h.sdkCallHandler.Verify()
	require.NoError(t, err, "sdkCallHandler should be called as expected")
}
