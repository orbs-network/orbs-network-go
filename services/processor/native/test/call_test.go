package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCallUnknownContract(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithUnknownContract().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}

func TestCallUnknownMethod(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithUnknownMethod().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}
