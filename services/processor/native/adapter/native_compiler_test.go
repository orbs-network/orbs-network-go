package adapter

import (
	"testing"
	"github.com/orbs-network/orbs-network-go/test/contracts"
	"os"
	"github.com/stretchr/testify/require"
	"fmt"
	"time"
	"io/ioutil"
	"strings"
)

const counterContractStartFrom = 100

func TestCompileCodeWithExistingArtifacts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping compilation of contracts in short mode")
	}

	code := string(contracts.SourceCodeForCounter(counterContractStartFrom))
	tmpDir := createTempTestDir(t)
	defer os.RemoveAll(tmpDir)

	t.Log("Build fresh artifacts")

	sourceFilePath, err := writeSourceCodeToDisk("testPrefix", code, tmpDir)
	require.NoError(t, err, "write to disk should succeed")
	require.FileExists(t, sourceFilePath, "file should exist")

	compilationStartTime := time.Now().UnixNano()
	soFilePath, err := buildSharedObject("testPrefix", sourceFilePath, tmpDir)
	require.NoError(t, err, "compilation should succeed")
	require.FileExists(t, soFilePath, "file should exist")
	compilationTimeMs := (time.Now().UnixNano() - compilationStartTime) / 1000000
	t.Logf("Compilation time: %d ms", compilationTimeMs)

	t.Log("Simulate corrupted artifacts and rebuild")

	// simulate corrupt file that exists
	err = ioutil.WriteFile(sourceFilePath, []byte{0x01}, 0644)
	require.NoError(t, err)
	require.Equal(t, int64(1), getFileSize(sourceFilePath), "file size should match")

	// simulate corrupt file that exists
	err = ioutil.WriteFile(soFilePath, []byte{0x01}, 0644)
	require.NoError(t, err)
	require.Equal(t, int64(1), getFileSize(soFilePath), "file size should match")

	sourceFilePath, err = writeSourceCodeToDisk("prefix", code, tmpDir)
	require.NoError(t, err, "write to disk should succeed")
	require.FileExists(t, sourceFilePath, "file should exist")
	require.NotEqual(t, int64(1), getFileSize(sourceFilePath), "file size should not match")

	compilationStartTime = time.Now().UnixNano()
	soFilePath, err = buildSharedObject("testPrefix", sourceFilePath, tmpDir)
	require.NoError(t, err, "compilation should succeed")
	require.FileExists(t, soFilePath, "file should exist")
	require.NotEqual(t, int64(1), getFileSize(soFilePath), "file size should not match")
	compilationTimeMs = (time.Now().UnixNano() - compilationStartTime) / 1000000
	t.Logf("Compilation time: %d ms", compilationTimeMs)

	t.Log("Load artifact")

	contractInfo, err := loadSharedObject(soFilePath)
	require.NoError(t, err, "load should succeed")
	require.NotNil(t, contractInfo, "loaded object should not be nil")
	require.Equal(t, fmt.Sprintf("CounterFrom%d", counterContractStartFrom), contractInfo.Name, "loaded object should be valid")

	t.Log("Try to rebuild already loaded artifact")

	compilationStartTime = time.Now().UnixNano()
	soFilePath, err = buildSharedObject("testPrefix", sourceFilePath, tmpDir)
	require.NoError(t, err, "compilation should succeed")
	require.FileExists(t, soFilePath, "file should exist")
	require.NotEqual(t, int64(1), getFileSize(soFilePath), "file size should not match")
	t.Logf("Compilation time: %d ms", compilationTimeMs)

	contractInfo, err = loadSharedObject(soFilePath)
	require.NoError(t, err, "load should succeed")
	require.NotNil(t, contractInfo, "loaded object should not be nil")
	require.Equal(t, fmt.Sprintf("CounterFrom%d", counterContractStartFrom), contractInfo.Name, "loaded object should be valid")
}

func createTempTestDir(t *testing.T) string {
	prefix := strings.Replace(t.Name(), "/", "__", -1)
	tmpDir, err := ioutil.TempDir("/tmp", prefix)
	if err != nil {
		panic("could not create temp dir for test")
	}
	return tmpDir
}

func getFileSize(filePath string) int64 {
	fi, err := os.Stat(filePath)
	if err != nil {
		panic("could not get file size")
	}
	return fi.Size()
}