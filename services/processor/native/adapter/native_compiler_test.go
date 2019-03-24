// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

//+build !nonativecompiler

package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"github.com/orbs-network/orbs-network-go/test/contracts/counter_mock"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

const COUNTER_CONTRACT_START_FROM = 100

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

	sourceFilePath, err := writeSourceCodeToDisk("testPrefix", code, tmpDir)
	require.NoError(t, err, "write to disk should succeed")
	require.FileExists(t, sourceFilePath, "file should exist")

	compilationStartTime := time.Now().UnixNano()
	soFilePath, err := buildSharedObject(ctx, "testPrefix", sourceFilePath, tmpDir)
	require.NoError(t, err, "compilation should succeed")
	require.FileExists(t, soFilePath, "file should exist")
	compilationTimeMs := (time.Now().UnixNano() - compilationStartTime) / 1000000
	t.Logf("Compilation time: %d ms", compilationTimeMs)

	t.Log("Simulate corrupted artifacts and rebuild")

	// simulate corrupt file that exists
	err = ioutil.WriteFile(sourceFilePath, []byte{0x01}, 0600)
	require.NoError(t, err)
	require.Equal(t, int64(1), getFileSize(sourceFilePath), "file size should match")

	// simulate corrupt file that exists
	err = ioutil.WriteFile(soFilePath, []byte{0x01}, 0600)
	require.NoError(t, err)
	require.Equal(t, int64(1), getFileSize(soFilePath), "file size should match")

	sourceFilePath, err = writeSourceCodeToDisk("testPrefix", code, tmpDir)
	require.NoError(t, err, "write to disk should succeed")
	require.FileExists(t, sourceFilePath, "file should exist")
	require.NotEqual(t, int64(1), getFileSize(sourceFilePath), "file size should not match")

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
