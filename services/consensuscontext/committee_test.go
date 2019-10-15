// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
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
