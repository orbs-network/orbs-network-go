package test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCallWithUnknownContractFails(t *testing.T) {
	h := newHarness()

	input := processCallInput().WithUnknownContract().Build()
	_, err := h.service.ProcessCall(input)
	require.Error(t, err, "call should fail")
}
