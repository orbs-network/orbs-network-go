package e2e

import (
	"github.com/orbs-network/orbs-client-sdk-go/codec"
	"github.com/stretchr/testify/require"
	"testing"
)

func (h *Harness) GetBlock(blockHeight uint64) (*codec.GetBlockResponse, error) {
	return h.client.GetBlock(blockHeight)
}

func TestGetBlock(t *testing.T) {
	h := NewAppHarness()
	h.WaitUntilTransactionPoolIsReady(t)

	blockResponse, err := h.GetBlock(1)
	require.NoError(t, err)
	require.EqualValues(t, 1, blockResponse.BlockHeight)
}
