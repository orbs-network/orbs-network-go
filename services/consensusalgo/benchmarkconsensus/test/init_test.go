package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"testing"
)

func TestInit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(true)
		h.expectHandlerRegistrations()
		h.createService(ctx)
		h.verifyHandlerRegistrations(t)
	})
}
