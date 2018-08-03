package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type harness struct {
	service services.Processor
}

func newHarness() *harness {
	sdkCallHandler := &handlers.MockContractSdkCallHandler{}

	service := native.NewNativeProcessor()
	service.RegisterContractSdkCallHandler(sdkCallHandler)

	return &harness{
		service: service,
	}
}
