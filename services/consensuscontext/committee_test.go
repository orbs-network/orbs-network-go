package consensuscontext

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConsensusContextCommittee_WrongInputs(t *testing.T) {
	tests := []struct {
		name                   string
		orderedCommittee []primitives.NodeAddress
		committeeMembers []primitives.NodeAddress
		committeeWeights []primitives.Weight
	}{
		{"not enough weights", []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.Weight{1} },
		{"too much weights", []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.Weight{1, 2, 3}},
		{"not enough unordered committee", []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.NodeAddress{{0x01}}, []primitives.Weight{1, 2}},
		{"too much unordered committee", []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.NodeAddress{{0x01}, {0x02}, {0x03}}, []primitives.Weight{1, 2}},
		{"not enough ordered committee", []primitives.NodeAddress{{0x01}}, []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.Weight{1, 2}},
		{"too much ordered committee", []primitives.NodeAddress{{0x01}, {0x02}, {0x03}}, []primitives.NodeAddress{{0x01}, {0x02}}, []primitives.Weight{1, 2}},
	}

	for i := range tests {
		cTest := tests[i]
		t.Run(cTest.name, func(t *testing.T) {
			_, err := orderCommitteeWeights(cTest.orderedCommittee, cTest.committeeMembers, cTest.committeeWeights)
			require.Error(t, err, "should fail with non equal lengths")
			require.Contains(t, err.Error(), "order weights failed sizes don't match")
		})
	}
}

func TestConsensusContextCommittee_AddressesMismatch(t *testing.T) {
	orderedCommittee := []primitives.NodeAddress {{0x01}, {0x02}, {0x03}}
	committeeMembers := []primitives.NodeAddress {{0x01}, {0x02}, {0x04}}
	committeeWeights := []primitives.Weight {1, 2, 3}

	_, err := orderCommitteeWeights(orderedCommittee, committeeMembers, committeeWeights)
	require.Error(t, err, "should fail with address mismatch")
	require.Contains(t, err.Error(), "order weights failed committee and ordered don't have same addresses")
}

func TestConsensusContextCommittee_SimpleHappyFlow(t *testing.T) {
	orderedCommittee := []primitives.NodeAddress {{0x01}, {0x02}, {0x03}}
	committeeMembers := []primitives.NodeAddress {{0x01}, {0x03}, {0x02}}
	committeeWeights := []primitives.Weight {1, 2, 3}

	orderedWeights, err := orderCommitteeWeights(orderedCommittee, committeeMembers, committeeWeights)
	require.NoError(t, err, "should succeed")
	require.EqualValues(t, []primitives.Weight {1, 3, 2}, orderedWeights)
}
