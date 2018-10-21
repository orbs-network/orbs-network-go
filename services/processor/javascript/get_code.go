package javascript

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

func (s *service) retrieveContractCodeFromRepository(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName primitives.ContractName) (string, error) {
	// 1. try artifact cache
	code := s.getContractFromRepository(contractName)
	if code != "" {
		return code, nil
	}

	// 2. try deployable code from state
	codeBytes, err := s.callGetCodeOfDeploymentSystemContract(ctx, executionContextId, contractName)
	if err != nil {
		return "", err
	}

	code = string(codeBytes)
	s.addContractToRepository(contractName, code)
	s.logger.Info("loaded deployable contract successfully", log.Stringable("contract", contractName))

	return code, nil
}

func (s *service) callGetCodeOfDeploymentSystemContract(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName primitives.ContractName) ([]byte, error) {
	handler := s.getContractSdkHandler()
	if handler == nil {
		return nil, errors.New("ContractSdkCallHandler has not registered yet")
	}

	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT.Name)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_GET_CODE.Name)

	output, err := handler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: native.SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:        "serviceName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemContractName),
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:        "methodName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemMethodName),
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:       "inputArgs",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: argsToMethodArgumentArray(string(contractName)).Raw(),
			}).Build(),
		},
		PermissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	})
	if err != nil {
		return nil, err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	methodArgumentArray := protocol.MethodArgumentArrayReader(output.OutputArguments[0].BytesValue())
	argIterator := methodArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesValue() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	return arg0.BytesValue(), nil
}

func argsToMethodArgumentArray(args ...interface{}) *protocol.MethodArgumentArray {
	res := []*protocol.MethodArgumentBuilder{}
	for _, arg := range args {
		switch arg.(type) {
		case uint32:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "uint32", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.(uint32)})
		case uint64:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "uint64", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.(uint64)})
		case string:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "string", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.(string)})
		case []byte:
			res = append(res, &protocol.MethodArgumentBuilder{Name: "bytes", Type: protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.([]byte)})
		}
	}
	return (&protocol.MethodArgumentArrayBuilder{Arguments: res}).Build()
}
