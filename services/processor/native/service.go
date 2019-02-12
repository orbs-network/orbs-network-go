package native

import (
	"context"
	"fmt"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
	"time"
)

var LogTag = log.Service("processor-native")

type service struct {
	logger     log.BasicLogger
	compiler   adapter.Compiler
	sdkHandler handlers.ContractSdkCallHandler

	contracts struct {
		sync.RWMutex
		instances     map[string]*types.ContractInstance
		deployedCache map[string]*sdkContext.ContractInfo
	}

	metrics *metrics
}

type metrics struct {
	deployedContracts       *metric.Gauge
	processCallTime         *metric.Histogram
	contractCompilationTime *metric.Histogram
}

func getMetrics(m metric.Factory) *metrics {
	return &metrics{
		deployedContracts:       m.NewGauge("Processor.Native.DeployedContractsNumber"),
		processCallTime:         m.NewLatency("Processor.Native.ProcessCallTime", 10*time.Second),
		contractCompilationTime: m.NewLatency("Processor.Native.ContractCompilationTime", 10*time.Second),
	}
}

func NewNativeProcessor(compiler adapter.Compiler, logger log.BasicLogger, metricFactory metric.Factory) services.Processor {
	s := &service{
		compiler: compiler,
		logger:   logger.WithTags(LogTag),
		metrics:  getMetrics(metricFactory),
	}

	s.contracts.instances = initializePreBuiltContractInstances()
	s.contracts.deployedCache = make(map[string]*sdkContext.ContractInfo)

	return s
}

// runs once on system initialization (called by the virtual machine constructor)
func (s *service) RegisterContractSdkCallHandler(handler handlers.ContractSdkCallHandler) {
	s.sdkHandler = handler
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

	// setup context for the contract sdk
	sdkContext.PushContext(sdkContext.ContextId(input.ContextId), s, contractInfo.Permission)
	defer sdkContext.PopContext(sdkContext.ContextId(input.ContextId))

	// get the method and check permissions
	contractInstance, methodInstance, err := s.retrieveContractAndMethodInstances(string(input.ContractName), string(input.MethodName), input.CallingPermissionScope)
	if err != nil {
		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: s.createMethodOutputArgsWithString(err.Error()),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_INPUT,
		}, err
	}

	start := time.Now()
	defer s.metrics.processCallTime.RecordSince(start)

	// execute
	functionNameForErrors := fmt.Sprintf("%s.%s", input.ContractName, input.MethodName)
	outputArgs, contractErr, err := s.processMethodCall(input.ContextId, contractInstance, methodInstance, input.InputArgumentArray, functionNameForErrors)
	if outputArgs == nil {
		outputArgs = (&protocol.ArgumentArrayBuilder{}).Build()
	}
	if err != nil {
		logger.Info("contract execution failed", log.Error(err))

		return &services.ProcessCallOutput{
			// TODO(https://github.com/orbs-network/orbs-spec/issues/97): do we need to remove system errors from OutputArguments?
			OutputArgumentArray: s.createMethodOutputArgsWithString(err.Error()),
			CallResult:          protocol.EXECUTION_RESULT_ERROR_INPUT,
		}, err
	}

	// result
	callResult := protocol.EXECUTION_RESULT_SUCCESS
	if contractErr != nil {
		logger.Info("contract returned error", log.Error(contractErr))

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

func (s *service) getContractInstance(contractName string) *types.ContractInstance {
	s.contracts.RLock()
	defer s.contracts.RUnlock()

	return s.contracts.instances[contractName]
}

func (s *service) addContractInstance(contractName string, instance *types.ContractInstance) {
	s.contracts.Lock()
	defer s.contracts.Unlock()

	s.contracts.instances[contractName] = instance
}

func (s *service) getDeployedContractInfoFromCache(contractName string) *sdkContext.ContractInfo {
	s.contracts.RLock()
	defer s.contracts.RUnlock()

	return s.contracts.deployedCache[contractName]
}

func (s *service) addDeployedContractInfoToCache(contractName string, contractInfo *sdkContext.ContractInfo) {
	s.contracts.Lock()
	defer s.contracts.Unlock()

	s.contracts.deployedCache[contractName] = contractInfo
}
