package consensuscontext

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"sort"
)

func (s *service) RequestOrderingCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	return s.RequestValidationCommittee(ctx, input)
}

func toPublicKeys(nodes map[string]config.FederationNode) []primitives.Ed25519PublicKey {
	keys := make([]primitives.Ed25519PublicKey, len(nodes))
	i := 0
	for _, value := range nodes {
		keys[i] = value.NodePublicKey()
		i++
	}
	return keys
}

func (s *service) RequestValidationCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	federationNodes := s.config.FederationNodes(uint64(input.BlockHeight))
	federationNodesPublicKeys := toPublicKeys(federationNodes)
	committeeSize := calculateCommitteeSize(input.MaxCommitteeSize, s.config.ConsensusMinimumCommitteeSize(), uint32(len(federationNodesPublicKeys)))
	indices, err := chooseRandomCommitteeIndices(committeeSize, input.RandomSeed, federationNodesPublicKeys)
	if err != nil {
		return nil, err
	}

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

func calculateCommitteeSize(requestedCommitteeSize uint32, minimumCommitteeSize uint32, federationSize uint32) uint32 {

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

// See https://github.com/orbs-network/orbs-spec/issues/111
func chooseRandomCommitteeIndices(committeeSize uint32, randomSeed uint64, nodes []primitives.Ed25519PublicKey) ([]uint32, error) {

	type gradedIndex struct {
		grade uint64
		index uint32
	}

	seedBytes := []byte(fmt.Sprintf("%x", randomSeed))

	grades := make([]*gradedIndex, len(nodes))

	i := 0
	for _, node := range nodes {

		// Reputation per node is presently not implemented so it is constant
		reputation := uint64(1)

		hashInput := make([]byte, len(seedBytes)+len(node))
		copy(hashInput, seedBytes)
		copy(hashInput[len(seedBytes):], node)
		nodeHash := hash.CalcSha256(hashInput)
		nodeHash4LSB := nodeHash[len(nodeHash)-4:]
		nodeHash4LSBInt := binary.LittleEndian.Uint32(nodeHash4LSB)
		grades[i] = &gradedIndex{
			grade: uint64(nodeHash4LSBInt) * reputation,
			index: uint32(i),
		}
		i++
	}
	// Descending order
	sort.Slice(grades, func(i, j int) bool {
		return grades[i].grade > grades[j].grade
	})

	indices := make([]uint32, committeeSize)
	for i := 0; i < int(committeeSize); i++ {
		indices[i] = grades[i].index
	}
	return indices, nil
}
