package filesystem

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestGetFileSizeGoodFile(t *testing.T) {
	ex, err := os.Executable()
	require.NoError(t, err, "os function failed")
	file, err := os.OpenFile(ex, os.O_RDWR, 0600)
	require.NoError(t, err, "should be able to read the current executing file size")

	size, err := getBlockFileSize(file)

	require.NoError(t, err, "should be able to read the current executing file size")
	require.Condition(t, func() bool { return size > 0 }, "size should be read okay for the current executing file")
}

func TestGetFileSizeBadFile(t *testing.T) {
	ex, err := os.Executable()
	require.NoError(t, err, "os function failed")
	file, err := os.OpenFile(ex, os.O_RDWR, 0600)
	require.NoError(t, err, "should be able to read the current executing file size")

	file.Close()
	_, errOfReadSize := getBlockFileSize(file)

	require.Error(t, errOfReadSize, "should not able to read size of closed file handle")
}
