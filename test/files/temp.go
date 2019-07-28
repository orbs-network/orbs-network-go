package files

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func NewTempFileWithContent(t testing.TB, content string) string {
	file, err := ioutil.TempFile("", "*")
	require.NoError(t, err, "failed creating config file")
	_, err = file.Write([]byte(content))
	require.NoError(t, err, "failed writing to config file")
	require.NoError(t, file.Close())
	return file.Name()
}

func RemoveSilently(file string) {
	_ = os.Remove(file)
}
