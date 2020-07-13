// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	"fmt"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var LogTag = log.Service("processor-native")

type Repository interface {
	ContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error)
}

type service struct {
	logger     log.Logger
	config     config.NativeProcessorConfig
	sdkHandler handlers.ContractSdkCallHandler

	cache *contractCache

	repository          Repository
	compilingRepository *CompilingRepository //TODO remove when refactor is done

	metrics *metrics
}

type metrics struct {
	processCallTime *metric.HistogramTimeDiff
}

func getMetrics(m metric.Factory) *metrics {
	return &metrics{
		processCallTime: m.NewLatency("Processor.Native.ProcessCallTime.Millis", 10*time.Second),
	}
}

func NewNativeProcessor(compiler adapter.Compiler, config config.NativeProcessorConfig, parentLogger log.Logger, metricFactory metric.Factory) services.Processor {
	logger := parentLogger.WithTags(LogTag)

	compilingRepository := NewCompilingRepository(compiler, config, parentLogger, metricFactory)
	compositeRepository := &CompositeRepository{Nested: []Repository{repository.NewPrebuilt(), compilingRepository}}

	return &service{
		repository:          compositeRepository,
		compilingRepository: compilingRepository,
		config:              config,
		logger:              logger,
		metrics:             getMetrics(metricFactory),
		cache:               newContractCache(),
	}
}

func NewProcessorWithContractRepository(repo Repository, config config.NativeProcessorConfig, parentLogger log.Logger, metricFactory metric.Factory) services.Processor {
	logger := parentLogger.WithTags(LogTag)
	compositeRepository := &CompositeRepository{Nested: []Repository{repository.NewPrebuilt(), repo}}

	return &service{
		repository: compositeRepository,
		config:     config,
		logger:     logger,
		metrics:    getMetrics(metricFactory),
		cache:      newContractCache(),
	}
}

// runs once on system initialization (called by the virtual machine constructor)
func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	s.sdkHandler = handler
	if s.compilingRepository != nil {
		s.compilingRepository.SetSdkHandler(handler)
	}
}

func (s *service) ProcessCall(ctx context.Context, input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	// retrieve code
	contractInfo, err := s.retrieveContractInfo(ctx, input.ContextId, string(input.ContractName))
	if err != nil {
		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: createMethodOutputArgsWithString(err.Error()),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED,
		}, err
	}

	// get the method and check permissions
	contractInstance, methodInstance, err := s.retrieveContractAndMethodInstances(contractInfo, string(input.ContractName), string(input.MethodName), input.CallingPermissionScope)
	if err != nil {
		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: createMethodOutputArgsWithString(err.Error()),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_INPUT,
		}, err
	}

	// setup context for the contract sdk
	sdkContext.PushContext(sdkContext.ContextId(input.ContextId), sdk.NewSDK(s.sdkHandler, s.config), contractInfo.Permission)
	defer sdkContext.PopContext(sdkContext.ContextId(input.ContextId))

	start := time.Now()
	defer s.metrics.processCallTime.RecordSince(start)

	// execute
	logger.Info("processor executing contract", log.Stringable("contract", input.ContractName), log.Stringable("method", input.MethodName))

	functionNameForErrors := fmt.Sprintf("%s.%s", input.ContractName, input.MethodName)
	outputArgs, contractErr, err := processMethodCall(input.ContextId, contractInstance, methodInstance, input.InputArgumentArray, functionNameForErrors)
	if outputArgs == nil {
		outputArgs = protocol.ArgumentsArrayEmpty()
	}
	if err != nil {
		logger.Info("contract execution failed", log.Stringable("contract", input.ContractName), log.Stringable("method", input.MethodName), log.Error(err))

		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: createMethodOutputArgsWithString(err.Error()),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_INPUT,
		}, err
	}

	// result
	callResult := protocol.EXECUTION_RESULT_SUCCESS
	if contractErr != nil {
		logger.Info("contract returned error", log.Stringable("contract", input.ContractName), log.Stringable("method", input.MethodName), log.Error(contractErr))

		callResult = protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	}
	return &services.ProcessCallOutput{
		OutputArgumentArray: outputArgs,
		CallResult:          callResult,
	}, contractErr
}

func (s *service) GetContractInfo(ctx context.Context, input *services.GetContractInfoInput) (*services.GetContractInfoOutput, error) {
	// retrieve code
	contractInfo, err := s.retrieveContractInfo(ctx, input.ContextId, string(input.ContractName))
	if err != nil {
		return nil, err
	}

	// result
	return &services.GetContractInfoOutput{
		PermissionScope: protocol.ExecutionPermissionScope(contractInfo.Permission),
	}, nil
}

func (s *service) retrieveContractAndMethodInstances(contractInfo *sdkContext.ContractInfo, contractName string, methodName string, permissionScope protocol.ExecutionPermissionScope) (*types.ContractInstance, types.MethodInstance, error) {
	contractInstance, err := s.getContractInstance(contractInfo, contractName)
	if err != nil {
		return nil, nil, errors.Errorf("error creating contract instance for contract %s", contractName)
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

func (s *service) retrieveContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	contractInfo := s.cache.infoByName(contractName)
	if contractInfo != nil {
		return contractInfo, nil
	}

	contractInfo, err := s.repository.ContractInfo(ctx, executionContextId, contractName)
	if err != nil {
		return nil, err
	}
	if contractInfo == nil {
		return nil, errors.Errorf("Contract %s was not found", contractName)
	}

	s.cache.addInfo(contractName, contractInfo)
	return contractInfo, err
}

func (s *service) getContractInstance(contractInfo *sdkContext.ContractInfo, contractName string) (*types.ContractInstance, error) {
	contractInstance := s.cache.instanceByNam(contractName)
	if contractInstance != nil {
		return contractInstance, nil
	}

	contractInstance, err := types.NewContractInstance(contractInfo)
	if err != nil {
		return nil, err
	}
	s.cache.addInstance(contractName, contractInstance)
	return contractInstance, nil
}

func (c *contractCache) infoByName(contractName string) *sdkContext.ContractInfo {
	c.RLock()
	defer c.RUnlock()

	return c.contractInfo[contractName]
}

func (c *contractCache) addInfo(contractName string, contractInfo *sdkContext.ContractInfo) {
	c.Lock()
	defer c.Unlock()

	c.contractInfo[contractName] = contractInfo
}

func (c *contractCache) instanceByNam(contractName string) *types.ContractInstance {
	c.RLock()
	defer c.RUnlock()

	return c.contractInstances[contractName]
}

func (c *contractCache) addInstance(contractName string, contractInstance *types.ContractInstance) {
	c.Lock()
	defer c.Unlock()

	c.contractInstances[contractName] = contractInstance
}

type contractCache struct {
	sync.RWMutex
	contractInfo      map[string]*sdkContext.ContractInfo
	contractInstances map[string]*types.ContractInstance
}

func newContractCache() *contractCache {
	return &contractCache{
		contractInfo:      make(map[string]*sdkContext.ContractInfo),
		contractInstances: make(map[string]*types.ContractInstance),
	}
}
