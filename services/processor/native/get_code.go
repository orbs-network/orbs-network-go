package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

func (s *service) initializePreBuiltRepositoryContractInstances(sdkHandler handlers.ContractSdkCallHandler) map[primitives.ContractName]types.Contract {
	preBuiltRepository := make(map[primitives.ContractName]types.Contract)
	for _, contract := range repository.Contracts {
		preBuiltRepository[contract.Name] = contract.InitSingleton(types.NewBaseContract(
			&stateSdk{sdkHandler, contract.Permission},
			&serviceSdk{sdkHandler, contract.Permission},
		))
	}
	return preBuiltRepository
}

func (s *service) retrieveContractAndMethodInfoFromRepository(executionContextId types.Context, contractName primitives.ContractName, methodName primitives.MethodName) (*types.ContractInfo, *types.MethodInfo, error) {
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

func (s *service) retrieveContractInfoFromRepository(executionContextId types.Context, contractName primitives.ContractName) (*types.ContractInfo, error) {
	// try pre-built repository first
	contract, found := repository.Contracts[contractName]
	if found {
		return &contract, nil
	}

	// try state for deployable second
	// TODO: artifact cache - no need to access state if an artifact is built
	return s.retrieveDeployableContractInfoFromState(executionContextId, contractName)
}

func (s *service) retrieveDeployableContractInfoFromState(executionContextId types.Context, contractName primitives.ContractName) (*types.ContractInfo, error) {
	return s.callGetDeploymentCodeSystemContract(executionContextId, contractName)
}

func (s *service) callGetDeploymentCodeSystemContract(executionContextId types.Context, contractName primitives.ContractName) (*types.ContractInfo, error) {
	handler := s.getContractSdkHandler()
	if handler == nil {
		return nil, errors.New("ContractSdkCallHandler has not registered yet")
	}

	_, err := handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:        "serviceName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(deployments.CONTRACT.Name),
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:        "methodName",
				Type:        protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(deployments.METHOD_GET_CODE.Name),
			}).Build(),
		},
		PermissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	})
	return nil, err
}
