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
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"time"
)

var LogTag = log.Service("processor-native")

type Repository interface {
	ContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error)
	RetrieveContractAndMethodInstances(contractName string, methodName string, permissionScope protocol.ExecutionPermissionScope) (contractInstance *types.ContractInstance, methodInstance types.MethodInstance, err error)
	SetSdkHandler(handler handlers.ContractSdkCallHandler)
}

type service struct {
	logger     log.Logger
	config     config.NativeProcessorConfig
	sdkHandler handlers.ContractSdkCallHandler

	repository Repository

	metrics *metrics
}

type metrics struct {
	processCallTime *metric.Histogram
}

func getMetrics(m metric.Factory) *metrics {
	return &metrics{
		processCallTime: m.NewLatency("Processor.Native.ProcessCallTime.Millis", 10*time.Second),
	}
}

func NewNativeProcessor(compiler adapter.Compiler, config config.NativeProcessorConfig, parentLogger log.Logger, metricFactory metric.Factory) services.Processor {
	logger := parentLogger.WithTags(LogTag)

	repository := &compositeRepository{
		compiler:                compiler,
		logger:                  logger,
		sanitizer:               createSanitizer(),
		deployedContracts:       metricFactory.NewGauge("Processor.Native.DeployedContracts.Count"),
		contractCompilationTime: metricFactory.NewLatency("Processor.Native.ContractCompilationTime.Millis", 10*time.Second),
	}
	repository.contracts.instances = initializePreBuiltContractInstances()
	repository.contracts.deployedCache = make(map[string]*sdkContext.ContractInfo)
	s := &service{
		repository: repository,
		config:     config,
		logger:     logger,
		metrics:    getMetrics(metricFactory),
	}

	return s
}

// runs once on system initialization (called by the virtual machine constructor)
func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	s.sdkHandler = handler
	s.repository.SetSdkHandler(handler)
}

func (s *service) ProcessCall(ctx context.Context, input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	// retrieve code
	contractInfo, err := s.retrieveContractInfo(ctx, input.ContextId, string(input.ContractName))
	if err != nil {
		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: s.createMethodOutputArgsWithString(err.Error()),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_CONTRACT_NOT_DEPLOYED,
		}, err
	}

	// get the method and check permissions
	contractInstance, methodInstance, err := s.repository.RetrieveContractAndMethodInstances(string(input.ContractName), string(input.MethodName), input.CallingPermissionScope)
	if err != nil {
		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: s.createMethodOutputArgsWithString(err.Error()),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_INPUT,
		}, err
	}

	// setup context for the contract sdk
	sdkContext.PushContext(sdkContext.ContextId(input.ContextId), s, contractInfo.Permission)
	defer sdkContext.PopContext(sdkContext.ContextId(input.ContextId))

	start := time.Now()
	defer s.metrics.processCallTime.RecordSince(start)

	// execute
	logger.Info("processor executing contract", log.Stringable("contract", input.ContractName), log.Stringable("method", input.MethodName))

	functionNameForErrors := fmt.Sprintf("%s.%s", input.ContractName, input.MethodName)
	outputArgs, contractErr, err := s.processMethodCall(input.ContextId, contractInstance, methodInstance, input.InputArgumentArray, functionNameForErrors)
	if outputArgs == nil {
		outputArgs = (&protocol.ArgumentArrayBuilder{}).Build()
	}
	if err != nil {
		logger.Info("contract execution failed", log.Stringable("contract", input.ContractName), log.Stringable("method", input.MethodName), log.Error(err))

		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: s.createMethodOutputArgsWithString(err.Error()),
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
