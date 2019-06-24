package test

import (
	"context"
	"github.com/orbs-network/go-mock"
)

type FaultySigner struct {
	mock.Mock
}

func (c *FaultySigner) Sign(ctx context.Context, input []byte) ([]byte, error) {
	call := c.Called(input)
	return call.Get(0).([]byte), call.Error(1)
}
