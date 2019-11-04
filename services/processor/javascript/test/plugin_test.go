//+build !race
// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"testing"
)

const DUMMY_PLUGIN_BIN = "services/processor/javascript/test/dummy_plugin.bin"

func TestProcessCall_WithLoadablePluginSucceeds(t *testing.T) {
	BuildDummyPlugin("services/processor/plugins/dummy/", DUMMY_PLUGIN_BIN)
	defer RemoveDummyPlugin(DUMMY_PLUGIN_BIN)

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, DummyPluginPath(DUMMY_PLUGIN_BIN))
			input := processCallInput().WithDeployableCounterContract(contracts.MOCK_COUNTER_CONTRACT_START_FROM).Build()
			codeOutput := builders.ArgumentsArray([]byte(contracts.JavaScriptSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM)))
			h.expectSdkCallMadeWithServiceCallMethod(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_CODE, builders.ArgumentsArray(string(input.ContractName)), codeOutput, nil)

			output, err := h.service.ProcessCall(ctx, input)
			require.NoError(t, err, "call should succeed")
			require.Equal(t, contracts.MOCK_COUNTER_CONTRACT_START_FROM, output.OutputArgumentArray.ArgumentsIterator().NextArguments().Uint64Value(), "call return value should be counter value")

			t.Log("First call should getCode for compilation")
			h.verifySdkCallMade(t)

			output, err = h.service.ProcessCall(ctx, input)
			require.NoError(t, err, "call should succeed")
			require.Equal(t, contracts.MOCK_COUNTER_CONTRACT_START_FROM, output.OutputArgumentArray.ArgumentsIterator().NextArguments().Uint64Value(), "call return value should be counter value")

			t.Log("Make sure second call does not getCode again")
			h.verifySdkCallMade(t)
		})
	})
}

func TestProcessCall_WithoutLoadablePlugin(t *testing.T) {
	RemoveDummyPlugin(DUMMY_PLUGIN_BIN)

	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, "")
			input := processCallInput().WithDeployableCounterContract(contracts.MOCK_COUNTER_CONTRACT_START_FROM).Build()
			codeOutput := builders.ArgumentsArray([]byte(contracts.JavaScriptSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM)))
			h.expectSdkCallMadeWithServiceCallMethod(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_CODE, builders.ArgumentsArray(string(input.ContractName)), codeOutput, nil)

			_, err := h.service.ProcessCall(ctx, input)
			require.EqualError(t, err, "JS processor is not implemented")
			h.verifySdkCallMade(t)
		})
	})
}
