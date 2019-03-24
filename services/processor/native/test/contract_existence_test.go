// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCall_WithUnknownContractFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		input := processCallInput().WithUnknownContract().Build()
		h.expectSdkCallMadeWithServiceCallMethod(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_CODE, builders.ArgumentsArray(string(input.ContractName)), nil, errors.New("code not found error"))

		_, err := h.service.ProcessCall(ctx, input)
		require.Error(t, err, "call should fail")

		h.verifySdkCallMade(t)
	})
}

func TestGetContractInfo_WithUnknownContractFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		input := getContractInfoInput().WithUnknownContract().Build()
		h.expectSdkCallMadeWithServiceCallMethod(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_CODE, builders.ArgumentsArray(string(input.ContractName)), nil, errors.New("code not found error"))

		_, err := h.service.GetContractInfo(ctx, input)
		require.Error(t, err, "GetContractInfo should fail")

		h.verifySdkCallMade(t)
	})
}

func TestProcessCall_WithDeployableContractThatCompiles(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		input := processCallInput().WithDeployableCounterContract(contracts.MOCK_COUNTER_CONTRACT_START_FROM).Build()
		codeOutput := builders.ArgumentsArray([]byte(contracts.NativeSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM)))
		h.expectSdkCallMadeWithServiceCallMethod(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_CODE, builders.ArgumentsArray(string(input.ContractName)), codeOutput, nil)

		output, err := h.service.ProcessCall(ctx, input)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, contracts.MOCK_COUNTER_CONTRACT_START_FROM, output.OutputArgumentArray.ArgumentsIterator().NextArguments().Uint64Value(), "call return value should be counter value")

		t.Log("First call (not compiled) should getCode for compilation")
		h.verifySdkCallMade(t)

		output, err = h.service.ProcessCall(ctx, input)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, contracts.MOCK_COUNTER_CONTRACT_START_FROM, output.OutputArgumentArray.ArgumentsIterator().NextArguments().Uint64Value(), "call return value should be counter value")

		t.Log("Make sure second call (already compiled) does not getCode again")
		h.verifySdkCallMade(t)
	})
}
