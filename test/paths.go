package test

import (
	"github.com/orbs-network/orbs-network-go/config"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func CreateTempDirForTest(t *testing.T) string {
	prefix := strings.Replace(t.Name(), "/", "__", -1)
	dir := filepath.Join(config.GetCurrentSourceFileDirPath(), "_tmp")
	os.MkdirAll(dir, 0700)
	tmpDir, err := ioutil.TempDir(dir, prefix)
	if err != nil {
		panic("could not create temp dir for test")
	}
	return tmpDir
}
