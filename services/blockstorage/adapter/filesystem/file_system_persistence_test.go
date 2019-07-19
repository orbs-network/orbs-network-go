package filesystem

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func TestGetFileSizeGoodFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	file, err := ioutil.TempFile("", "")
	require.NoError(t, err, "failed to create a temp file")
	defer func() { _ = os.Remove(file.Name()) }()

	size, err := getBlockFileSize(file)

	require.NoError(t, err, "should be able to read file size of an open file")
	require.EqualValues(t, 0, size, "size should be read okay for the current executing file")

	// write some data
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	n, err := file.Write(data)
	require.NoError(t, err, "failed to write some bytes to temp file")
	require.EqualValues(t, len(data), n, "expected written size to equal bytes array size")

	size, err = getBlockFileSize(file)
	require.NoError(t, err, "should be able to read file size of an open file")
	require.EqualValues(t, n, size, "file size does not match the number of bytes written to it")
}

func TestGetFileSizeBadFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	file, err := ioutil.TempFile("", "")
	require.NoError(t, err, "failed to create a temp file")
	defer func() { _ = os.Remove(file.Name()) }()

	err = file.Close()
	require.NoError(t, err, "expected closing of the temp file to succeed")

	_, errOfReadSize := getBlockFileSize(file)

	require.Error(t, errOfReadSize, "should not able to read size of closed file handle")
}
