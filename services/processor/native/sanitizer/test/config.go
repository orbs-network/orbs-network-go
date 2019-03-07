package test

import "github.com/orbs-network/orbs-network-go/services/processor/native/sanitizer"

func SanitizerConfigForTests() *sanitizer.SanitizerConfig {
	return &sanitizer.SanitizerConfig{
		ImportWhitelist: map[string]bool{
			`"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"`:       true,
			`"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"`: true,
		},
	}
}
