package test

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCallUnknownContractFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithUnknownContract().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}

func TestCallUnknownMethodFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithUnknownMethod().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}

func TestCallExternalMethodFromAnotherServiceSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithExternalMethod().WithDifferentCallingService().Build()

	_, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
}

func TestCallInternalMethodFromSameServiceSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithInternalMethod().WithSameCallingService().Build()

	_, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
}

func TestCallInternalMethodFromAnotherServiceFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithInternalMethod().WithDifferentCallingService().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}

func TestCallInternalMethodFromAnotherServiceUnderSystemPermissionsSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithInternalMethod().WithDifferentCallingService().WithSystemPermissions().Build()

	_, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
}

func TestCallWriteMethodWithWriteAccessSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithExternalWriteMethod().WithWriteAccess().Build()

	_, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
}

func TestCallWriteMethodWithoutWriteAccessFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithExternalWriteMethod().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}
