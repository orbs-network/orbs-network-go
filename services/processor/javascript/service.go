// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.
//
// +build javascript

package javascript

import (
	"context"
	"fmt"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/processor"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"sync"
)

var LogTag = log.Service("processor-javascript")

type service struct {
	logger log.Logger
	config config.JavascriptProcessorConfig

	mutex               *sync.RWMutex
	sdkHandler          handlers.ContractSdkCallHandler
	contractsUnderMutex map[primitives.ContractName]string

	worker func(handler sdkContext.SdkHandler) processor.StatelessProcessor
}

func NewJavaScriptProcessor(logger log.Logger, config config.JavascriptProcessorConfig) services.Processor {
	var worker func(handler sdkContext.SdkHandler) processor.StatelessProcessor
	var err error

	if config.ExperimentalExternalProcessorPluginPath() != "" {
		worker, err = loadPlugin(config.ExperimentalExternalProcessorPluginPath())
		if err != nil {
			panic(fmt.Sprintf("Could not load plugin: %s", err))
		}
	} else {
		worker = DefaultWorker
	}

	return &service{
		logger:              logger.WithTags(LogTag),
		mutex:               &sync.RWMutex{},
		contractsUnderMutex: make(map[primitives.ContractName]string),
		worker:              worker,
		config:              config,
	}
}

// runs once on system initialization (called by the virtual machine constructor)
func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.sdkHandler = handler
}

func (s *service) ProcessCall(ctx context.Context, input *services.ProcessCallInput) (*services.ProcessCallOutput, error) {
	// retrieve code
	code, err := s.retrieveContractCodeFromRepository(ctx, input.ContextId, input.ContractName)
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArgumentArray: (&protocol.ArgumentArrayBuilder{}).Build(),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// execute
	outputArgs, contractErr, err := s.processMethodCall(input.ContextId, code, input.MethodName, input.InputArgumentArray)
	if outputArgs == nil {
		outputArgs = (&protocol.ArgumentArrayBuilder{}).Build()
	}
	if err != nil {
		return &services.ProcessCallOutput{
			OutputArgumentArray: outputArgs,
			CallResult:          protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, err
	}

	// result
	callResult := protocol.EXECUTION_RESULT_SUCCESS
	if contractErr != nil {
		callResult = protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT
	}
	return &services.ProcessCallOutput{
		OutputArgumentArray: outputArgs,
		CallResult:          callResult,
	}, contractErr
}

func (s *service) GetContractInfo(ctx context.Context, input *services.GetContractInfoInput) (*services.GetContractInfoOutput, error) {
	panic("Not implemented")
}

func (s *service) getContractSdkHandler() handlers.ContractSdkCallHandler {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.sdkHandler
}

func (s *service) getContractFromRepository(contractName primitives.ContractName) string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.contractsUnderMutex == nil {
		return ""
	}
	return s.contractsUnderMutex[contractName]
}

func (s *service) addContractToRepository(contractName primitives.ContractName, code string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.contractsUnderMutex == nil {
		return
	}
	s.contractsUnderMutex[contractName] = code
}
