package test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCallUnknownContractFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithUnknownContract().Build()

	_, err := h.service.ProcessCall(call)
	require.Error(t, err, "call should fail")
}

func TestCallUnknownMethodFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithUnknownMethod().Build()

	_, err := h.service.ProcessCall(call)
	require.Error(t, err, "call should fail")
}

func TestCallExternalMethodFromAnotherServiceSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithExternalMethod().WithDifferentCallingService().Build()

	_, err := h.service.ProcessCall(call)
	require.NoError(t, err, "call should succeed")
}

func TestCallInternalMethodFromSameServiceSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithInternalMethod().WithSameCallingService().Build()

	_, err := h.service.ProcessCall(call)
	require.NoError(t, err, "call should succeed")
}

func TestCallInternalMethodFromAnotherServiceFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithInternalMethod().WithDifferentCallingService().Build()

	_, err := h.service.ProcessCall(call)
	require.Error(t, err, "call should fail")
}

func TestCallInternalMethodFromAnotherServiceUnderSystemPermissionsSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithInternalMethod().WithDifferentCallingService().WithSystemPermissions().Build()

	_, err := h.service.ProcessCall(call)
	require.NoError(t, err, "call should succeed")
}

func TestCallWriteMethodWithWriteAccessSucceeds(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithExternalWriteMethod().WithWriteAccess().Build()
	h.expectSdkCallMadeWithStateWrite()

	_, err := h.service.ProcessCall(call)
	require.NoError(t, err, "call should succeed")
}

func TestCallWriteMethodWithoutWriteAccessFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithExternalWriteMethod().Build()

	_, err := h.service.ProcessCall(call)
	require.Error(t, err, "call should fail")
}
