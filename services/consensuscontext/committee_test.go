// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommitteeSizeVSTotalNodesCount(t *testing.T) {

	TOTAL_VALIDATORS_SIZE := uint32(10)
	MINIMUM_COMMITTEE_SIZE := TOTAL_VALIDATORS_SIZE - 2

	testCases := []struct {
		description            string
		requestedCommitteeSize uint32
		totalValidatorsSize    uint32
		expectedCommitteeSize  uint32
	}{
		{"Requested committee smaller than total validators", TOTAL_VALIDATORS_SIZE - 1, TOTAL_VALIDATORS_SIZE, TOTAL_VALIDATORS_SIZE - 1},
		{"Requested committee same size as total validators", TOTAL_VALIDATORS_SIZE, TOTAL_VALIDATORS_SIZE, TOTAL_VALIDATORS_SIZE},
		{"Requested committee larger than total validators", TOTAL_VALIDATORS_SIZE + 1, TOTAL_VALIDATORS_SIZE, TOTAL_VALIDATORS_SIZE},
		{"Requested committee less than minimum", MINIMUM_COMMITTEE_SIZE - 1, TOTAL_VALIDATORS_SIZE, MINIMUM_COMMITTEE_SIZE},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			actualCommitteeSize := calculateCommitteeSize(testCase.requestedCommitteeSize, MINIMUM_COMMITTEE_SIZE, testCase.totalValidatorsSize)
			require.Equal(t, testCase.expectedCommitteeSize, actualCommitteeSize,
				"Expected committee size is %d but the calculated committee size is %d",
				testCase.expectedCommitteeSize, actualCommitteeSize)
		})
	}
}

func TestChooseRandomCommitteeIndices(t *testing.T) {
	nodeAddresses := testKeys.NodeAddressesForTests()
	input := &services.RequestCommitteeInput{
		CurrentBlockHeight: 1,
		RandomSeed:         123456789,
		MaxCommitteeSize:   5,
	}

	t.Run("Receive same number of indices as requested", func(t *testing.T) {
		indices, err := chooseRandomCommitteeIndices(input.MaxCommitteeSize, input.RandomSeed, nodeAddresses)
		if err != nil {
			t.Error(err)
		}
		indicesLen := uint32(len(indices))
		require.Equal(t, input.MaxCommitteeSize, indicesLen, "Expected to receive %d indices but got %d", input.MaxCommitteeSize, indicesLen)
	})

	t.Run("Receive unique indices", func(t *testing.T) {
		indices, err := chooseRandomCommitteeIndices(input.MaxCommitteeSize, input.RandomSeed, nodeAddresses)
		if err != nil {
			t.Error(err)
		}
		uniqueIndices := unique(indices)
		uniqueIndicesLen := uint32(len(uniqueIndices))
		require.Equal(t, input.MaxCommitteeSize, uniqueIndicesLen, "Expected to receive %d unique indices but got %d", input.MaxCommitteeSize, uniqueIndicesLen)
	})

	t.Run("Receive below number of indices requested", func(t *testing.T) {
		nodeSubset := nodeAddresses[:3]
		indices, err := chooseRandomCommitteeIndices(input.MaxCommitteeSize, input.RandomSeed, nodeSubset)
		if err != nil {
			t.Error(err)
		}
		indicesLen := uint32(len(indices))
		require.EqualValues(t, len(nodeSubset), indicesLen, "Expected to receive %d indices but got %d", len(nodeSubset), indicesLen)
		require.True(t, indicesLen < input.MaxCommitteeSize, "Received %d indices that should be below %d", indicesLen, input.MaxCommitteeSize)
	})

}

func unique(input []uint32) []uint32 {
	u := make([]uint32, 0, len(input))
	m := make(map[uint32]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}
	return u
}
