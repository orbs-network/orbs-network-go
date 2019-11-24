package test

import (
	"github.com/orbs-network/orbs-network-go/test"
	"os"
	"testing"
)

type Config interface {
	ProcessorArtifactPath() string
	ProcessorPerformWarmUpCompilation() bool
}

func NewConfigWithTempDir(t *testing.T) (config Config, cleanupFunc func()) {
	tmpDir := test.CreateTempDirForTest(t)
	cleanupFunc = func() {
		_ = os.RemoveAll(tmpDir)
	}
	config = &tempDirConfig{ArtifactPath: tmpDir}
	return config, cleanupFunc
}

type tempDirConfig struct {
	ArtifactPath string
}

func (c *tempDirConfig) ProcessorPerformWarmUpCompilation() bool {
	return true
}

func (c *tempDirConfig) ProcessorArtifactPath() string {
	return c.ArtifactPath
}
