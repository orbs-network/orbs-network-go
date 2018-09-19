package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkAddress_GetSignerAddressWithoutContext(t *testing.T) {
	h := newHarness()

	_, err := h.handleSdkCall(999, native.SDK_OPERATION_NAME_ADDRESS, "getSignerAddress")
	require.Error(t, err, "handleSdkCall should fail")
}
