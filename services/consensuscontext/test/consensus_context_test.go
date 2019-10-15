// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRequestOrderingCommittee_ReturnsMaxCommitteeSizeMembers_WhenCommitteeSizeIsSmallerThanGenesisValidatorsSize(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		h := newHarness(harness.Logger, false)
		blockHeight := primitives.BlockHeight(1)
		genesisValidatorsSize := len(h.config.GenesisValidatorNodes())
		h.expectVirtualMachineToReturnGenesisCommittee()

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

}

func TestRequestOrderingCommittee_ReturnsAllGenesisValidators_WhenCommitteeSizeIsLargerThanGenesisValidatorsSize(t *testing.T) {
	with.Logging(t, func(harness *with.LoggingHarness) {
		h := newHarness(harness.Logger, false)
		blockHeight := primitives.BlockHeight(1)
		genesisValidatorsSize := len(h.config.GenesisValidatorNodes())
		h.expectVirtualMachineToReturnGenesisCommittee()

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
