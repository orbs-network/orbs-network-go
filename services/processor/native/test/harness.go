package test

import (
	"bytes"
	"encoding/binary"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/harness/services/processor/native/adapter"
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
	log := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))

	compiler := adapter.NewFakeCompiler()
	compiler.ProvideFakeContract(contracts.MockForCounter(), string(contracts.NativeSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM)))

	sdkCallHandler := &handlers.MockContractSdkCallHandler{}

	registry := metric.NewRegistry()

	service := native.NewNativeProcessor(compiler, log, registry)
	service.RegisterContractSdkCallHandler(sdkCallHandler)

	return &harness{
		sdkCallHandler: sdkCallHandler,
		service:        service,
	}
}

func (h *harness) expectSdkCallMadeWithStateRead(expectedKey []byte, returnValue []byte) {
	stateReadCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == native.SDK_OPERATION_NAME_STATE &&
			input.MethodName == "read" &&
			len(input.InputArguments) == 1 &&
			(expectedKey == nil || bytes.Equal(input.InputArguments[0].BytesValue(), expectedKey))
	}

	readReturn := &handlers.HandleSdkCallOutput{
		OutputArguments: builders.MethodArguments(returnValue),
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.State, method equals read and 1 arg matches", stateReadCallMatcher)).Return(readReturn, nil).Times(1)
}

func (h *harness) expectSdkCallMadeWithStateWrite(expectedKey []byte, expectedValue []byte) {
	stateWriteCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == native.SDK_OPERATION_NAME_STATE &&
			input.MethodName == "write" &&
			len(input.InputArguments) == 2 &&
			(expectedKey == nil || bytes.Equal(input.InputArguments[0].BytesValue(), expectedKey)) &&
			(expectedValue == nil || bytes.Equal(input.InputArguments[1].BytesValue(), expectedValue))
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.State, method equals write and 2 args match", stateWriteCallMatcher)).Return(nil, nil).Times(1)
}

func (h *harness) expectSdkCallMadeWithServiceCallMethod(expectedContractName string, expectedMethodName string, expectedArgArray *protocol.MethodArgumentArray, returnArgArray *protocol.MethodArgumentArray, returnError error) {
	serviceCallMethodCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == native.SDK_OPERATION_NAME_SERVICE &&
			input.MethodName == "callMethod" &&
			len(input.InputArguments) == 3 &&
			input.InputArguments[0].StringValue() == expectedContractName &&
			input.InputArguments[1].StringValue() == expectedMethodName &&
			bytes.Equal(input.InputArguments[2].BytesValue(), expectedArgArray.Raw())
	}

	var returnOutput *handlers.HandleSdkCallOutput
	if returnArgArray != nil {
		returnOutput = &handlers.HandleSdkCallOutput{
			OutputArguments: builders.MethodArguments(returnArgArray.Raw()),
		}
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.Service, method equals callMethod and 3 args match", serviceCallMethodCallMatcher)).Return(returnOutput, returnError).Times(1)
}

func (h *harness) expectSdkCallMadeWithAddressGetCaller(returnAddress []byte) {
	addressGetCallerCallMatcher := func(i interface{}) bool {
		input, ok := i.(*handlers.HandleSdkCallInput)
		return ok &&
			input.OperationName == native.SDK_OPERATION_NAME_ADDRESS &&
			input.MethodName == "getCallerAddress"
	}

	returnOutput := &handlers.HandleSdkCallOutput{
		OutputArguments: builders.MethodArguments(returnAddress),
	}

	h.sdkCallHandler.When("HandleSdkCall", mock.Any, mock.AnyIf("Contract equals Sdk.Address, method equals getCallerAddress and 1 arg match", addressGetCallerCallMatcher)).Return(returnOutput, nil).Times(1)
}

func (h *harness) verifySdkCallMade(t *testing.T) {
	_, err := h.sdkCallHandler.Verify()
	require.NoError(t, err, "sdkCallHandler should be called as expected")
}

func uint64ToBytes(num uint64) []byte {
	res := make([]byte, 8)
	binary.LittleEndian.PutUint64(res, num)
	return res
}
