package test

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRequestOrderingCommittee(t *testing.T) {
	h := newHarness()
	blockHeight := primitives.BlockHeight(1)
	federationSize := len(h.config.FederationNodes(uint64(blockHeight)))

	t.Run("if MaxCommitteeSize <= federationSize, then return MaxCommitteeSize members", func(t *testing.T) {
		input := &services.RequestCommitteeInput{
			CurrentBlockHeight: blockHeight,
			RandomSeed:         0,
			MaxCommitteeSize:   uint32(federationSize),
		}
		output, err := h.service.RequestOrderingCommittee(context.Background(), input)
		if err != nil {
			t.Error(err)
		}
		actualFederationSize := len(output.NodeAddresses)
		require.Equal(t, federationSize, actualFederationSize, "expected committee size is %d but got %d", federationSize, actualFederationSize)
	})
	t.Run("if MaxCommitteeSize > federationSize, then return all federation members (federationSize)", func(t *testing.T) {
		input := &services.RequestCommitteeInput{
			CurrentBlockHeight: blockHeight,
			RandomSeed:         0,
			MaxCommitteeSize:   uint32(federationSize + 1),
		}
		output, err := h.service.RequestOrderingCommittee(context.Background(), input)
		if err != nil {
			t.Error(err)
		}
		actualFederationSize := len(output.NodeAddresses)
		require.Equal(t, federationSize, actualFederationSize, "expected committee size is %d but got %d", federationSize, actualFederationSize)
	})
}

func TestCreateAndValidateTransactionsBlock(t *testing.T) {

}
