package testkit

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	benchmarkcontract "github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkContract"
	processorTests "github.com/orbs-network/orbs-network-go/services/processor/native/test"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessorCanCallContractWithManualRepository(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			cfg := &processorTests.NativeProcessorConfigForTests{}
			r := NewRepository()
			p := native.NewProcessorWithContractRepository(r, cfg, harness.Logger, metric.NewRegistry())

			r.Register(benchmarkcontract.CONTRACT_NAME, benchmarkcontract.PUBLIC, benchmarkcontract.SYSTEM, benchmarkcontract.EVENTS, sdkContext.PERMISSION_SCOPE_SERVICE)

			call := processorTests.ProcessCallInput().WithMethod("BenchmarkContract", "add").WithArgs(uint64(12), uint64(27)).Build()

			output, err := p.ProcessCall(ctx, call)
			require.NoError(t, err, "call should succeed")
			require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
			require.Equal(t, builders.ArgumentsArray(uint64(12+27)), output.OutputArgumentArray, "call return args should be equal")
		})
	})
}
