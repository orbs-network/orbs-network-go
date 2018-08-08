package test

import (
	"testing"
)

func TestInit(t *testing.T) {
	h := newHarness()
	h.verifyHandlerRegistrations(t)
}
