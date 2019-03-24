// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func testFileContents(t *testing.T, filename string, expected string) {
	f, _ := os.Open(filename)
	defer f.Close()

	contents, _ := ioutil.ReadAll(f)
	require.EqualValues(t, []byte(expected), contents)
}

func TestNewTruncatingFileWriterWithNoDefault(t *testing.T) {
	tmp, err := ioutil.TempFile("/tmp", "truncatingFileWriter")
	require.NoError(t, err)
	defer tmp.Close()

	w := NewTruncatingFileWriter(tmp)
	w.Write([]byte("hello"))
	testFileContents(t, tmp.Name(), "hello")

	w.Truncate()

	w.Write([]byte("something else"))
	testFileContents(t, tmp.Name(), "something else")
}

func TestNewTruncatingFileWriterWithAutoTruncate(t *testing.T) {
	tmp, err := ioutil.TempFile("/tmp", "truncatingFileWriter")
	require.NoError(t, err)
	defer tmp.Close()

	w := NewTruncatingFileWriter(tmp, 1*time.Millisecond)
	w.Write([]byte("hello"))
	testFileContents(t, tmp.Name(), "hello")

	time.Sleep(1 * time.Millisecond)

	w.Write([]byte("something else"))
	testFileContents(t, tmp.Name(), "something else")

	time.Sleep(1 * time.Millisecond)

	w.Write([]byte("another thing"))
	testFileContents(t, tmp.Name(), "another thing")
}
