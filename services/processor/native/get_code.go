package native

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

func initializePreBuiltRepositoryContractInstances(sdkHandler handlers.ContractSdkCallHandler) map[string]sdk.ContractInstance {
	preBuiltRepository := make(map[string]sdk.ContractInstance)
	for _, contractInfo := range repository.PreBuiltContracts {
		preBuiltRepository[contractInfo.Name] = initializeContractInstance(contractInfo, sdkHandler)
	}
	return preBuiltRepository
}

func initializeContractInstance(contractInfo *sdk.ContractInfo, sdkHandler handlers.ContractSdkCallHandler) sdk.ContractInstance {
	return contractInfo.InitSingleton(sdk.NewBaseContract(
		&stateSdk{sdkHandler, protocol.ExecutionPermissionScope(contractInfo.Permission)},
		&serviceSdk{sdkHandler, protocol.ExecutionPermissionScope(contractInfo.Permission)},
	))
}

func (s *service) retrieveContractAndMethodInfoFromRepository(executionContextId sdk.Context, contractName string, methodName string) (*sdk.ContractInfo, *sdk.MethodInfo, error) {
	contract, err := s.retrieveContractInfoFromRepository(executionContextId, contractName)
	if err != nil {
		return nil, nil, err
	}
	method, found := contract.Methods[methodName]
	if !found {
		return nil, nil, errors.Errorf("method '%s' not found in contract '%s'", methodName, contractName)
	}
	return contract, &method, nil
}

func (s *service) retrieveContractInfoFromRepository(executionContextId sdk.Context, contractName string) (*sdk.ContractInfo, error) {
	// 1. try pre-built repository
	contractInfo, found := repository.PreBuiltContracts[contractName]
	if found {
		return contractInfo, nil
	}

	// 2. try deployable artifact cache (if already compiled)
	contractInfo = s.getDeployableContractInfoFromRepository(contractName)
	if contractInfo != nil {
		return contractInfo, nil
	}

	// 3. try deployable code from state (if not yet compiled)
	return s.retrieveDeployableContractInfoFromState(executionContextId, contractName)
}

const artifactsPath = "/tmp/orbs/native-processor/" // TODO: move to config?

func (s *service) retrieveDeployableContractInfoFromState(executionContextId sdk.Context, contractName string) (*sdk.ContractInfo, error) {
	codeBytes, err := s.callGetCodeOfDeploymentSystemContract(executionContextId, contractName)
	if err != nil {
		return nil, err
	}

	code, err := sanitizeDeployedSourceCode(string(codeBytes))
	if err != nil {
		return nil, errors.Wrapf(err, "source code for contract '%s' failed security sandbox audit", contractName)
	}

	newContractInfo, err := compileAndLoadDeployedSourceCode(code, artifactsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "compilation of deployable contract '%s' failed", contractName)
	}
	if newContractInfo == nil {
		return nil, errors.Errorf("compilation and load of deployable contract '%s' did not return a valid symbol", contractName)
	}

	sdkHandler := s.getContractSdkHandler()
	if sdkHandler == nil {
		return nil, errors.New("ContractSdkCallHandler has not registered yet")
	}
	contractInstance := initializeContractInstance(newContractInfo, sdkHandler)

	s.addContractInstanceToRepository(contractName, contractInstance)
	s.addDeployableContractInfoToRepository(contractName, newContractInfo) // must add after instance to avoid race (when somebody RunsMethod at same time)
	s.reporting.Info("compiled and loaded deployable contract successfully", log.String("contract", contractName))

	return newContractInfo, nil
}

func (s *service) callGetCodeOfDeploymentSystemContract(executionContextId sdk.Context, contractName string) ([]byte, error) {
	handler := s.getContractSdkHandler()
	if handler == nil {
		return nil, errors.New("ContractSdkCallHandler has not registered yet")
	}

	systemContractName := primitives.ContractName(deployments.CONTRACT.Name)
	systemMethodName := primitives.MethodName(deployments.METHOD_GET_CODE.Name)

	output, err := handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_SERVICE,
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
