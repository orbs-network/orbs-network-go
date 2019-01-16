package log

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
)

func testFileContents(t *testing.T, filename string, expected string) {
	f, _ := os.Open(filename)
	defer f.Close()

	contents, _ := ioutil.ReadAll(f)
	require.EqualValues(t, []byte(expected), contents)
}

func TestNewRotatingFileWriter(t *testing.T) {
	tmp, err := ioutil.TempFile("/tmp", "rotatingFileWriter")
	require.NoError(t, err)
	defer tmp.Close()

	w := NewRotatingFileWriter(tmp)
	w.Write([]byte("hello"))
	testFileContents(t, tmp.Name(), "hello")

	w.Rotate()

	w.Write([]byte("something else"))
	testFileContents(t, tmp.Name(), "something else")
}
