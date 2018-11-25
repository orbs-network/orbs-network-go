package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCommitteeSizeVSTotalNodesCount(t *testing.T) {

	federationSize := uint32(10)
	minimumCommitteeSize := federationSize - 2

	testCases := []struct {
		description            string
		requestedCommitteeSize uint32
		federationSize         uint32
		expectedCommitteeSize  uint32
	}{
		{"Requested committee smaller than federation", federationSize - 1, federationSize, federationSize - 1},
		{"Requested committee same size as federation", federationSize, federationSize, federationSize},
		{"Requested committee larger than federation", federationSize + 1, federationSize, federationSize},
		{"Requested committee less than minimum", minimumCommitteeSize - 1, federationSize, minimumCommitteeSize},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			actualCommitteeSize := calculateCommitteeSize(testCase.requestedCommitteeSize, minimumCommitteeSize, testCase.federationSize)
			require.Equal(t, testCase.expectedCommitteeSize, actualCommitteeSize,
				"Expected committee size is %d but the calculated committee size is %d",
				testCase.expectedCommitteeSize, actualCommitteeSize)
		})
	}
}

func TestChooseRandomCommitteeIndices(t *testing.T) {
	publicKeys := keys.Ed25519PublicKeysForTests()
	input := &services.RequestCommitteeInput{
		BlockHeight:      1,
		RandomSeed:       123456789,
		MaxCommitteeSize: 5,
	}

	t.Run("Receive same number of indices as requested", func(t *testing.T) {
		indices, err := chooseRandomCommitteeIndices(input.MaxCommitteeSize, input.RandomSeed, publicKeys)
		if err != nil {
			t.Error(err)
		}
		indicesLen := uint32(len(indices))
		require.Equal(t, input.MaxCommitteeSize, indicesLen, "Expected to receive %d indices but got %d", input.MaxCommitteeSize, indicesLen)
	})

	t.Run("Receive unique indices", func(t *testing.T) {
		indices, err := chooseRandomCommitteeIndices(input.MaxCommitteeSize, input.RandomSeed, publicKeys)
		if err != nil {
			t.Error(err)
		}
		uniqueIndices := unique(indices)
		uniqueIndicesLen := uint32(len(uniqueIndices))
		require.Equal(t, input.MaxCommitteeSize, uniqueIndicesLen, "Expected to receive %d unique indices but got %d", input.MaxCommitteeSize, uniqueIndicesLen)
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
