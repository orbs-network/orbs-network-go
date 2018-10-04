package sync

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewBlockSync(t *testing.T) {
	h := primitives.BlockHeight(10)
	sync := NewBlockSync(h)
	require.NotNil(t, sync, "block sync initialized")
	require.EqualValues(t, h, sync.lastBlockHeight, "block sync known height initialized correctly")
}
