package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type harness struct {
	service services.Processor
}

func newHarness() *harness {
	return &harness{
		service: native.NewNativeProcessor(),
	}
}
