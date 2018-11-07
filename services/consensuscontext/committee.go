package consensuscontext

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"sort"
)

func (s *service) RequestOrderingCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	federationNodes := s.config.FederationNodes(uint64(input.BlockHeight))
	federationNodesPublicKeys := toAscendingPublicKeys(federationNodes)
	committeeSize := CalculateCommitteeSize(input.MaxCommitteeSize, s.config.ConsensusMinimumCommitteeSize(), uint32(len(federationNodesPublicKeys)))
	indices := ChooseRandomCommitteeIndices(committeeSize, input.RandomSeed)

	committeePublicKeys := make([]primitives.Ed25519PublicKey, len(indices))
	for i, index := range indices {
		committeePublicKeys[i] = primitives.Ed25519PublicKey(federationNodesPublicKeys[int(index)])
	}

	res := &services.RequestCommitteeOutput{
		NodePublicKeys:           committeePublicKeys,
		NodeRandomSeedPublicKeys: nil,
	}

	return res, nil
}

func toAscendingPublicKeys(nodes map[string]config.FederationNode) []string {
	keys := make([]string, len(nodes))
	i := 0
	for key := range nodes {
		keys[i] = key
		i++
	}
	sort.Strings(keys)

	return keys
}

// TODO Pending a different impl if necessary
func (s *service) RequestValidationCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	return s.RequestOrderingCommittee(ctx, input)
}

func CalculateCommitteeSize(requestedCommitteeSize uint32, minimumCommitteeSize uint32, federationSize uint32) uint32 {

	if federationSize < minimumCommitteeSize {
		panic(fmt.Sprintf("config error: federation size %d cannot be less than minimum committee size %d", federationSize, minimumCommitteeSize))
	}

	if requestedCommitteeSize < minimumCommitteeSize {
		return minimumCommitteeSize
	}

	if requestedCommitteeSize > federationSize {
		return federationSize
	}
	return requestedCommitteeSize
}

// Smart algo!
func ChooseRandomCommitteeIndices(committeeSize uint32, randomSeed uint64) []uint32 {
	indices := make([]uint32, committeeSize)
	for i := 0; i < int(committeeSize); i++ {
		indices[i] = uint32(i)
	}
	return indices
}
