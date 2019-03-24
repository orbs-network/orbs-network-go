// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter/fake"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestContract_Compile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation of contracts in short mode")
	}

	t.Run("FakeCompiler", compileTest(aFakeCompiler))
	t.Run("NativeCompiler", compileTest(aNativeCompiler))
}

func compileTest(newHarness func(t *testing.T) *compilerContractHarness) func(*testing.T) {
	return func(t *testing.T) {
		h := newHarness(t)
		defer h.cleanup()

		// give the test one minute timeout to compile
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		t.Log("Compiling a valid contract")

		code := string(contracts.NativeSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM))
		compilationStartTime := time.Now().UnixNano()
		contractInfo, err := h.compiler.Compile(ctx, code)
		compilationTimeMs := (time.Now().UnixNano() - compilationStartTime) / 1000000
		t.Logf("Compilation time: %d ms", compilationTimeMs)

		require.NoError(t, err, "compile should succeed")
		require.NotNil(t, contractInfo, "loaded object should not be nil")

		// instantiate the "start()" function of the contract and call it
		contractInstance, err := types.NewContractInstance(contractInfo)
		require.NoError(t, err, "create contract instance should succeed")
		res := reflect.ValueOf(contractInstance.PublicMethods["start"]).Call([]reflect.Value{})
		require.Equal(t, contracts.MOCK_COUNTER_CONTRACT_START_FROM, res[0].Interface().(uint64), "result of calling start() should match")

		t.Log("Compiling an invalid contract")

		invalidCode := "invalid code example"
		_, err = h.compiler.Compile(ctx, invalidCode)
		require.Error(t, err, "compile should fail")
	}
}

type compilerContractHarness struct {
	compiler adapter.Compiler
	cleanup  func()
}

func aNativeCompiler(t *testing.T) *compilerContractHarness {
	tmpDir := test.CreateTempDirForTest(t)
	cfg := &hardcodedConfig{artifactPath: tmpDir}
	log := log.DefaultTestingLogger(t)
	compiler := adapter.NewNativeCompiler(cfg, log)
	return &compilerContractHarness{
		compiler: compiler,
		cleanup: func() {
			os.RemoveAll(tmpDir)
		},
	}
}

func aFakeCompiler(t *testing.T) *compilerContractHarness {
	compiler := fake.NewCompiler()
	code := string(contracts.NativeSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM))
	compiler.ProvideFakeContract(contracts.MockForCounter(), code)
	return &compilerContractHarness{
		compiler: compiler,
		cleanup:  func() {},
	}
}

type hardcodedConfig struct {
	artifactPath string
}

func (c *hardcodedConfig) ProcessorArtifactPath() string {
	return c.artifactPath
}
