// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

//+build !nonativecompiler
//+build !race

package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	adapterTest "github.com/orbs-network/orbs-network-go/services/processor/native/adapter/test"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/contracts/counter_mock"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
	"time"
)

const COUNTER_CONTRACT_START_FROM = 100

func TestCompileValidContract(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		if testing.Short() {
			t.Skip("Skipping compilation of contracts in short mode")
		}
		err := compilationOfMockCounterContract(t, parent)
		require.NoError(t, err)
	})
}

func TestCompileContractConcurrently(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		if testing.Short() {
			t.Skip("Skipping compilation of contracts in short mode")
		}

		const concurrentCount = 5
		doneChan := make(chan interface{}, concurrentCount)
		for i := 0; i < concurrentCount; i++ {
			go func() {
				err := compilationOfMockCounterContract(t, parent)
				doneChan <- err
			}()
		}

		for i := 0; i < concurrentCount; i++ {
			ctxErr := <-doneChan
			require.Nil(t, ctxErr, "expected concurrent contract compilation to succeed")
		}
	})
}

func TestCompileTimeout(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		if testing.Short() {
			t.Skip("Skipping compilation of contracts in short mode")
		}
		// hopefully enough time to start compiling, but not enough to finish
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		cfg, cleanup := adapterTest.NewConfigWithTempDir(t)
		defer cleanup()
		compiler := NewNativeCompiler(cfg, parent.Logger, metric.NewRegistry())

		code := string(contracts.NativeSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM))
		_, err := compiler.Compile(ctx, code)
		t.Log("Error : ", err)
		require.Error(t, err, "compile should fail")
		require.Regexp(t, regexp.MustCompile("deadline exceeded"), err.Error(), "message must hint timeout")
	})
}

func TestCompileInvalidContract(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		if testing.Short() {
			t.Skip("Skipping compilation of contracts in short mode")
		}
		// give the test one minute timeout to compile
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		cfg, cleanup := adapterTest.NewConfigWithTempDir(t)
		defer cleanup()
		compiler := NewNativeCompiler(cfg, parent.Logger, metric.NewRegistry())

		invalidCode := "package fail"
		_, err := compiler.Compile(ctx, invalidCode)
		t.Log("Error : ", err)
		require.Error(t, err, "compile should fail")
	})
}

func TestCompileCodeWithExistingArtifacts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation of contracts in short mode")
	}

	// give the test one minute timeout to compile
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	code := string(contracts.NativeSourceCodeForCounter(COUNTER_CONTRACT_START_FROM))
	tmpDir := test.CreateTempDirForTest(t)
	defer os.RemoveAll(tmpDir)

	t.Log("Build fresh artifacts")

	sourceFilePath, err := writeSourceCodeToDisk("testPrefix", []string{code}, tmpDir)
	require.NoError(t, err, "write to disk should succeed")
	require.NotZero(t, len(sourceFilePath))
	for _, path := range sourceFilePath {
		require.FileExists(t, path, "file should exist")
	}

	compilationStartTime := time.Now().UnixNano()
	soFilePath, err := buildSharedObject(ctx, "testPrefix", sourceFilePath, tmpDir)
	require.NoError(t, err, "compilation should succeed")
	require.FileExists(t, soFilePath, "file should exist")
	compilationTimeMs := (time.Now().UnixNano() - compilationStartTime) / 1000000
	t.Logf("Compilation time: %d ms", compilationTimeMs)

	t.Log("Simulate corrupted artifacts and rebuild")

	// simulate corrupt file that exists
	err = ioutil.WriteFile(sourceFilePath[0], []byte{0x01}, 0600)
	require.NoError(t, err)
	require.Equal(t, int64(1), getFileSize(sourceFilePath[0]), "file size should match")

	// simulate corrupt file that exists
	err = ioutil.WriteFile(soFilePath, []byte{0x01}, 0600)
	require.NoError(t, err)
	require.Equal(t, int64(1), getFileSize(soFilePath), "file size should match")

	sourceFilePath, err = writeSourceCodeToDisk("testPrefix", []string{code}, tmpDir)
	require.NoError(t, err, "write to disk should succeed")
	require.NotZero(t, len(sourceFilePath))
	for _, path := range sourceFilePath {
		require.FileExists(t, path, "file should exist")
	}
	require.NotEqual(t, int64(1), getFileSize(sourceFilePath[0]), "file size should not match")

	compilationStartTime = time.Now().UnixNano()
	soFilePath, err = buildSharedObject(ctx, "testPrefix", sourceFilePath, tmpDir)
	require.NoError(t, err, "compilation should succeed")
	require.FileExists(t, soFilePath, "file should exist")
	require.NotEqual(t, int64(1), getFileSize(soFilePath), "file size should not match")
	compilationTimeMs = (time.Now().UnixNano() - compilationStartTime) / 1000000
	t.Logf("Compilation time: %d ms", compilationTimeMs)

	t.Log("Load artifact")

	contractInfo, err := loadSharedObject(soFilePath)
	require.NoError(t, err, "load should succeed")
	require.NotNil(t, contractInfo, "loaded object should not be nil")
	require.Equal(t, len(counter_mock.PUBLIC), len(contractInfo.PublicMethods), "loaded object should be valid")

	t.Log("Try to rebuild already loaded artifact")

	compilationStartTime = time.Now().UnixNano()
	soFilePath, err = buildSharedObject(ctx, "testPrefix", sourceFilePath, tmpDir)
	require.NoError(t, err, "compilation should succeed")
	require.FileExists(t, soFilePath, "file should exist")
	require.NotEqual(t, int64(1), getFileSize(soFilePath), "file size should not match")
	compilationTimeMs = (time.Now().UnixNano() - compilationStartTime) / 1000000
	t.Logf("Compilation time: %d ms", compilationTimeMs)

	contractInfo, err = loadSharedObject(soFilePath)
	require.NoError(t, err, "load should succeed")
	require.NotNil(t, contractInfo, "loaded object should not be nil")
	require.Equal(t, len(counter_mock.PUBLIC), len(contractInfo.PublicMethods), "loaded object should be valid")
}

func getFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	if err != nil {
		panic("could not get file size")
	}
	return fi.Size()
}

func compilationOfMockCounterContract(t *testing.T, parent *with.LoggingHarness) error {
	// give the test one minute timeout to compile
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cfg, cleanup := adapterTest.NewConfigWithTempDir(t)
	defer cleanup()
	compiler := NewNativeCompiler(cfg, parent.Logger, metric.NewRegistry())
	code := string(contracts.NativeSourceCodeForCounter(contracts.MOCK_COUNTER_CONTRACT_START_FROM))
	_, err := compiler.Compile(ctx, code)
	return err
}
