// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package javascript

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/sdk"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
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

	systemContractName := primitives.ContractName(deployments_systemcontract.CONTRACT_NAME)
	systemMethodName := primitives.MethodName(deployments_systemcontract.METHOD_GET_CODE)

	output, err := handler.HandleSdkCall(ctx, &handlers.HandleSdkCallInput{
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
				BytesValue: builders.ArgumentsArray(string(contractName)).Raw(),
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
	argumentArray := protocol.ArgumentArrayReader(output.OutputArguments[0].BytesValue())
	argIterator := argumentArray.ArgumentsIterator()
	if !argIterator.HasNext() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	arg0 := argIterator.NextArguments()
	if !arg0.IsTypeBytesValue() {
		return nil, errors.Errorf("callMethod Sdk.Service of _Deployments.getCode returned corrupt output value")
	}
	return arg0.BytesValue(), nil
}
