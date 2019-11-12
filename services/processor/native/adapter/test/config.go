package test

import (
	"github.com/orbs-network/orbs-network-go/test"
	"os"
	"testing"
)

func NewConfigWithTempDir(t *testing.T) (config *HardcodedConfig, cleanupFunc func()) {
	tmpDir := test.CreateTempDirForTest(t)
	cleanupFunc = func() {
		os.RemoveAll(tmpDir)
	}
	config = &HardcodedConfig{ArtifactPath: tmpDir}
	return config, cleanupFunc
}

type HardcodedConfig struct {
	ArtifactPath string
}

func (c *HardcodedConfig) ProcessorPerformWarmUpCompilation() bool {
	return true
}

func (c *HardcodedConfig) ProcessorArtifactPath() string {
	return c.ArtifactPath
}
