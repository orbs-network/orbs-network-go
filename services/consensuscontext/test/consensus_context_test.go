package test

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRequestOrderingCommittee(t *testing.T) {
	h := newHarness(t)
	blockHeight := primitives.BlockHeight(1)
	genesisValidatorsSize := len(h.config.GenesisValidatorNodes(uint64(blockHeight)))

	t.Run("if MaxCommitteeSize <= genesisValidatorsSize, then return MaxCommitteeSize members", func(t *testing.T) {
		input := &services.RequestCommitteeInput{
			CurrentBlockHeight: blockHeight,
			RandomSeed:         0,
			MaxCommitteeSize:   uint32(genesisValidatorsSize),
		}
		output, err := h.service.RequestOrderingCommittee(context.Background(), input)
		if err != nil {
			t.Error(err)
		}
		actualNumberOfValidators := len(output.NodeAddresses)
		require.Equal(t, genesisValidatorsSize, actualNumberOfValidators, "expected committee size is %d but got %d", genesisValidatorsSize, actualNumberOfValidators)
	})
	t.Run("if MaxCommitteeSize > genesisValidatorsSize, then return all genesis validators (genesisValidatorsSize)", func(t *testing.T) {
		input := &services.RequestCommitteeInput{
			CurrentBlockHeight: blockHeight,
			RandomSeed:         0,
			MaxCommitteeSize:   uint32(genesisValidatorsSize + 1),
		}
		output, err := h.service.RequestOrderingCommittee(context.Background(), input)
		if err != nil {
			t.Error(err)
		}
		actualNumberOfValidators := len(output.NodeAddresses)
		require.Equal(t, genesisValidatorsSize, actualNumberOfValidators, "expected committee size is %d but got %d", genesisValidatorsSize, actualNumberOfValidators)
	})
}
