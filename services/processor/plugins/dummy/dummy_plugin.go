package main

import (
	"github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/services/processor"
)

func New(handler context.SdkHandler) processor.StatelessProcessor {
	return &dummyProcessor{
		handler,
	}
}
