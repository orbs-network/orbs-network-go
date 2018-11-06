package consensuscontext

import "github.com/orbs-network/orbs-spec/types/go/services"

func CalculateCommitteeSize(requestedCommitteeSize int, federationSize int) int {

	if requestedCommitteeSize > federationSize {
		return federationSize
	}
	return requestedCommitteeSize
}

// Smart algo!
func ChooseRandomCommitteeIndices(input *services.RequestCommitteeInput) []int {
	indices := make([]int, input.MaxCommitteeSize)
	for i := 0; i < int(input.MaxCommitteeSize); i++ {
		indices[i] = i
	}
	return indices
}
