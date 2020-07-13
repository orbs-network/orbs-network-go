// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package native

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"time"
)

type CompositeRepository struct {
	Nested []Repository
}

func (c *CompositeRepository) ContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	for _, repo := range c.Nested {
		contractInfo, err := repo.ContractInfo(ctx, executionContextId, contractName)
		if err != nil {
			return nil, err
		}
		if contractInfo != nil {
			return contractInfo, nil
		}
	}

	return nil, nil
}

func NewCompilingRepository(compiler adapter.Compiler, cfg config.NativeProcessorConfig, logger log.Logger, metricFactory metric.Factory) *CompilingRepository {
	compilingRepository := &CompilingRepository{
		compiler:                compiler,
		config:                  cfg,
		logger:                  logger.WithTags(log.Service("compiling-contract-repository")),
		sanitizer:               createSanitizer(),
		deployedContracts:       metricFactory.NewGauge("Processor.Native.DeployedContracts.Count"),
		contractCompilationTime: metricFactory.NewLatency("Processor.Native.ContractCompilationTime.Millis", 10*time.Second),
	}
	return compilingRepository
}

type CompilingRepository struct {
	compiler   adapter.Compiler
	sdkHandler handlers.ContractSdkCallHandler
	logger     log.Logger

	sanitizer *sanitizer.Sanitizer

	deployedContracts       *metric.Gauge
	processCallTime         *metric.HistogramTimeDiff
	contractCompilationTime *metric.HistogramTimeDiff
	config                  config.NativeProcessorConfig
}

func (r *CompilingRepository) SetSdkHandler(handler handlers.ContractSdkCallHandler) {
	r.sdkHandler = handler
}

func (r *CompilingRepository) ContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	return r.retrieveDeployedContractInfoFromState(ctx, executionContextId, contractName)
}

func (r *CompilingRepository) retrieveDeployedContractInfoFromState(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	start := time.Now()

	rawCodeFiles, err := r.getFullCodeOfDeploymentSystemContract(ctx, executionContextId, contractName)
	if err != nil {
		return nil, err
	}

	var code []string
	for _, rawCodeFile := range rawCodeFiles {
		sanitizedCode, err := r.sanitizeDeployedSourceCode(rawCodeFile)
		if err != nil {
			return nil, errors.Wrapf(err, "source code for contract '%s' failed security sandbox audit", contractName)
		}
		code = append(code, sanitizedCode)
	}

	// TODO(v1): replace with given wrapped given context
	ctx, cancel := context.WithTimeout(context.Background(), adapter.MAX_COMPILATION_TIME)
	defer cancel()

	newContractInfo, err := r.compiler.Compile(ctx, code...)
	if err != nil {
		return nil, errors.Wrapf(err, "compilation of deployable contract '%s' failed", contractName)
	}
	if newContractInfo == nil {
		return nil, errors.Errorf("compilation and load of deployable contract '%s' did not return a valid symbol", contractName)
	}

	r.logger.Info("compiled and loaded deployable contract successfully", log.String("contract", contractName))

	r.deployedContracts.Inc()
	r.contractCompilationTime.RecordSince(start)
	// only want to log meter on success (so this line is not under defer)

	return newContractInfo, nil
}

func (r *CompilingRepository) sanitizeDeployedSourceCode(code string) (string, error) {
	if r.config.ProcessorSanitizeDeployedContracts() {
		return r.sanitizer.Process(code)
	}

	return code, nil
}

func (r *CompilingRepository) getFullCodeOfDeploymentSystemContract(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) ([]string, error) {
	codeParts, err := r.getCodeParts(ctx, executionContextId, contractName)
	if err != nil {
		return nil, err
	}

	var results []string
	for i := uint32(0); i < codeParts; i++ {
		part, err := r.callGetCodeOfDeploymentSystemContract(ctx, executionContextId, contractName, i)
		if err != nil {
			return nil, err
		}
		results = append(results, part)
	}

	return results, nil
}

func (r *CompilingRepository) callGetCodeOfDeploymentSystemContract(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string, index uint32) (string, error) {
	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_GET_CODE_PART)
	inputArguments, err := protocol.ArgumentArrayFromNatives([]interface{}{contractName, index})
	if err != nil {
		panic(errors.Wrap(err, "input arguments"))
	}

	output, err := r.sdkHandler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: sdk.SDK_OPERATION_NAME_SERVICE,
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
				BytesValue: inputArguments.Raw(),
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

func (r *CompilingRepository) getCodeParts(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (uint32, error) {
	systemContractName := deployments_systemcontract.CONTRACT_NAME
	systemMethodName := deployments_systemcontract.METHOD_GET_CODE_PARTS
	inputArguments, err := protocol.ArgumentArrayFromNatives([]interface{}{contractName})
	if err != nil {
		panic(errors.Wrap(err, "input arguments"))
	}

	output, err := r.sdkHandler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: sdk.SDK_OPERATION_NAME_SERVICE,
		MethodName:    "callMethod",
		InputArguments: []*protocol.Argument{
			(&protocol.ArgumentBuilder{
				// serviceName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: systemContractName,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// methodName
				Type:        protocol.ARGUMENT_TYPE_STRING_VALUE,
				StringValue: systemMethodName,
			}).Build(),
			(&protocol.ArgumentBuilder{
				// inputArgs
				Type:       protocol.ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: inputArguments.Raw(),
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
