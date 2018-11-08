package test

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

/*

Questions:

1. What is weighted random sorting algo, and do we use reputation here
2. "minimal-block-delay-sec" - max wait time for tx? so should be called "max..."
3. metadata placeholder
4.


*/

func TestRequestOrderingCommittee(t *testing.T) {
	h := newHarness()
	blockHeight := primitives.BlockHeight(1)
	federationSize := len(h.config.FederationNodes(uint64(blockHeight)))

	t.Run("if MaxCommitteeSize <= federationSize, then return MaxCommitteeSize members", func(t *testing.T) {
		input := &services.RequestCommitteeInput{
			BlockHeight:      blockHeight,
			RandomSeed:       0,
			MaxCommitteeSize: uint32(federationSize),
		}
		output, err := h.service.RequestOrderingCommittee(context.Background(), input)
		if err != nil {
			t.Error(err)
		}
		actualFederationSize := len(output.NodePublicKeys)
		require.Equal(t, federationSize, actualFederationSize, "expected committee size is %d but got %d", federationSize, actualFederationSize)
	})
	t.Run("if MaxCommitteeSize > federationSize, then return all federation members (federationSize)", func(t *testing.T) {
		input := &services.RequestCommitteeInput{
			BlockHeight:      blockHeight,
			RandomSeed:       0,
			MaxCommitteeSize: uint32(federationSize + 1),
		}
		output, err := h.service.RequestOrderingCommittee(context.Background(), input)
		if err != nil {
			t.Error(err)
		}
		actualFederationSize := len(output.NodePublicKeys)
		require.Equal(t, federationSize, actualFederationSize, "expected committee size is %d but got %d", federationSize, actualFederationSize)
	})
}
