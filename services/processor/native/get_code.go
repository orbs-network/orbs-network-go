// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
	"time"
)

func initializePreBuiltContractInstances() map[string]*types.ContractInstance {
	res := make(map[string]*types.ContractInstance)
	for contractName, contractInfo := range repository.PreBuiltContracts {
		instance, err := types.NewContractInstance(contractInfo)
		if err == nil {
			res[contractName] = instance
		}
	}
	return res
}

type compositeRepository struct {
	compiler   adapter.Compiler
	sdkHandler handlers.ContractSdkCallHandler
	logger     log.Logger

	contracts struct {
		sync.RWMutex
		instances     map[string]*types.ContractInstance
		deployedCache map[string]*sdkContext.ContractInfo
	}
	sanitizer *sanitizer.Sanitizer

	deployedContracts       *metric.Gauge
	processCallTime         *metric.Histogram
	contractCompilationTime *metric.Histogram
}

func (r *compositeRepository) SetSdkHandler(handler handlers.ContractSdkCallHandler) {
	r.sdkHandler = handler
}

func (r *compositeRepository) ContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	// 1. try pre-built repository
	contractInfo, found := repository.PreBuiltContracts[contractName]
	if found {
		return contractInfo, nil
	}

	// 2. try deployed artifact cache (if already compiled)
	contractInfo = r.getDeployedContractInfoFromCache(contractName)
	if contractInfo != nil {
		return contractInfo, nil
	}

	// 3. try deployable code from state (if not yet compiled)
	return r.retrieveDeployedContractInfoFromState(ctx, executionContextId, contractName)
}

func (s *compositeRepository) ContractInstance(contractName string) *types.ContractInstance {
	s.contracts.RLock()
	defer s.contracts.RUnlock()

	return s.contracts.instances[contractName]
}

func (s *service) retrieveContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	return s.repository.ContractInfo(ctx, executionContextId, contractName)
}

func (s *compositeRepository) RetrieveContractAndMethodInstances(contractName string, methodName string, permissionScope protocol.ExecutionPermissionScope) (contractInstance *types.ContractInstance, methodInstance types.MethodInstance, err error) {
	contractInstance = s.ContractInstance(contractName)
	if contractInstance == nil {
		return nil, nil, errors.Errorf("contract instance not found for contract '%s'", contractName)
	}

	methodInstance, found := contractInstance.PublicMethods[methodName]
	if found {
		return contractInstance, methodInstance, nil
	}

	methodInstance, found = contractInstance.SystemMethods[methodName]
	if found {
		if permissionScope == protocol.PERMISSION_SCOPE_SYSTEM {
			return contractInstance, methodInstance, nil
		} else {
			return nil, nil, errors.Errorf("only system contracts can run method '%s'", methodName)
		}
	}

	return nil, nil, errors.Errorf("method '%s' not found on contract '%s'", methodName, contractName)
}

func (s *compositeRepository) retrieveDeployedContractInfoFromState(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	start := time.Now()

	rawCodeFiles, err := s.getFullCodeOfDeploymentSystemContract(ctx, executionContextId, contractName)
	if err != nil {
		return nil, err
	}

	var code []string

	for _, rawCodeFile := range rawCodeFiles {
		sanitizedCode, err := s.sanitizeDeployedSourceCode(rawCodeFile)
		if err != nil {
			return nil, errors.Wrapf(err, "source code for contract '%s' failed security sandbox audit", contractName)
		}
		code = append(code, sanitizedCode)
	}

	// TODO(v1): replace with given wrapped given context
	ctx, cancel := context.WithTimeout(context.Background(), adapter.MAX_COMPILATION_TIME)
	defer cancel()

	newContractInfo, err := s.compiler.Compile(ctx, code...)
	if err != nil {
		return nil, errors.Wrapf(err, "compilation of deployable contract '%s' failed", contractName)
	}
	if newContractInfo == nil {
		return nil, errors.Errorf("compilation and load of deployable contract '%s' did not return a valid symbol", contractName)
	}

	instance, err := types.NewContractInstance(newContractInfo)
	if err != nil {
		return nil, errors.Errorf("instance initialization of deployable contract '%s' failed", contractName)
	}
	s.addContractInstance(contractName, instance)
	s.addDeployedContractInfoToCache(contractName, newContractInfo) // must add after instance to avoid race (when somebody RunsMethod at same time)

	s.logger.Info("compiled and loaded deployable contract successfully", log.String("contract", contractName))

	s.deployedContracts.Inc()
	s.contractCompilationTime.RecordSince(start)
	// only want to log meter on success (so this line is not under defer)

	return newContractInfo, nil
}

func (s *compositeRepository) getFullCodeOfDeploymentSystemContract(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) ([]string, error) {
	codeParts, err := s.getCodeParts(ctx, executionContextId, contractName)
	if err != nil {
		return nil, err
	}

	var results []string
	for i := uint32(0); i < codeParts; i++ {
		part, err := s.callGetCodeOfDeploymentSystemContract(ctx, executionContextId, contractName, i)
		if err != nil {
			return nil, err
		}
		results = append(results, part)
	}

	return results, nil
}

func (s *compositeRepository) callGetCodeOfDeploymentSystemContract(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string, index uint32) (string, error) {
	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_GET_CODE_PART)

	output, err := s.sdkHandler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// serviceName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemContractName),
			}).Build(),
			(&protocol.ArgumentBuilder{
				// methodName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemMethodName),
			}).Build(),
			(&protocol.ArgumentBuilder{
				// inputArgs
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: argsToArgumentArray(string(contractName), index).Raw(),
			}).Build(),
		},
		PermissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	})
	if err != nil {
		return "", err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return "", errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	ArgumentArray := protocol.ArgumentArrayReader(output.OutputArguments[0].BytesValue())
	argIterator := ArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return "", errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesValue() {
		return "", errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	return string(arg0.BytesValue()), nil
}

func (s *compositeRepository) getCodeParts(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (uint32, error) {
	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_GET_CODE_PARTS)

	output, err := s.sdkHandler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// serviceName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemContractName),
			}).Build(),
			(&protocol.ArgumentBuilder{
				// methodName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: string(systemMethodName),
			}).Build(),
			(&protocol.ArgumentBuilder{
				// inputArgs
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: argsToArgumentArray(string(contractName)).Raw(),
			}).Build(),
		},
		PermissionScope: protocol.PERMISSION_SCOPE_SYSTEM,
	})
	if err != nil {
		return 0, err
	}

	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return 0, errors.Errorf("callMethod Sdk.Service of _Deployments.getCodeParts returned corrupt output value")
	}
	ArgumentArray := protocol.ArgumentArrayReader(output.OutputArguments[0].BytesValue())
	argIterator := ArgumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return 0, errors.Errorf("callMethod Sdk.Service of _Deployments.getCodeParts returned corrupt output value")
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeUint32Value() {
		return 0, errors.Errorf("callMethod Sdk.Service of _Deployments.getCodeParts returned corrupt output value")
	}

	return arg0.Uint32Value(), nil
}

func (s *compositeRepository) addContractInstance(contractName string, instance *types.ContractInstance) {
	s.contracts.Lock()
	defer s.contracts.Unlock()

	s.contracts.instances[contractName] = instance
}

func (s *compositeRepository) getDeployedContractInfoFromCache(contractName string) *sdkContext.ContractInfo {
	s.contracts.RLock()
	defer s.contracts.RUnlock()

	return s.contracts.deployedCache[contractName]
}

func (s *compositeRepository) addDeployedContractInfoToCache(contractName string, contractInfo *sdkContext.ContractInfo) {
	s.contracts.Lock()
	defer s.contracts.Unlock()

	s.contracts.deployedCache[contractName] = contractInfo
}
