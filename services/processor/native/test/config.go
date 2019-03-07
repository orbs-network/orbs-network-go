package test

type nativeProcessorConfigForTests struct {
}

func (c *nativeProcessorConfigForTests) ProcessorSanitizeDeployedContracts() bool {
	return false
}
