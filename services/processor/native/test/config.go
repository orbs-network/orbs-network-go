package test

import "github.com/orbs-network/orbs-spec/types/go/primitives"

type nativeProcessorConfigForTests struct {
}

func (c *nativeProcessorConfigForTests) ProcessorSanitizeDeployedContracts() bool {
	return false
}

func (c *nativeProcessorConfigForTests) VirtualChainId() primitives.VirtualChainId {
	return 42
}
