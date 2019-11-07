package test

type HardcodedConfig struct {
	ArtifactPath string
}

func (c *HardcodedConfig) ProcessorPerformWarmUpCompilation() bool {
	return true
}

func (c *HardcodedConfig) ProcessorArtifactPath() string {
	return c.ArtifactPath
}
