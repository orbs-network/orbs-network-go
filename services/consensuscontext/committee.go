// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"sort"
)

func (s *service) RequestOrderingCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	return s.RequestValidationCommittee(ctx, input)
}

func (s *service) RequestValidationCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	electedValidatorsAddresses, err := s.getElectedValidators(ctx, input.CurrentBlockHeight)
	if err != nil {
		return nil, err
	}

	committeeSize := calculateCommitteeSize(input.MaxCommitteeSize, s.config.LeanHelixConsensusMinimumCommitteeSize(), uint32(len(electedValidatorsAddresses)))
	logger.Info("Calculated committee size", logfields.BlockHeight(input.CurrentBlockHeight), log.Uint32("committee-size", committeeSize), log.Int("elected-validators-count", len(electedValidatorsAddresses)), log.Uint32("max-committee-size", input.MaxCommitteeSize))
	indices, err := chooseRandomCommitteeIndices(committeeSize, input.RandomSeed, electedValidatorsAddresses)
	if err != nil {
		return nil, err
	}

	committeeNodeAddresses := make([]primitives.NodeAddress, len(indices))
	for i, index := range indices {
		committeeNodeAddresses[i] = primitives.NodeAddress(electedValidatorsAddresses[int(index)])
	}

	res := &services.RequestCommitteeOutput{
		NodeAddresses:            committeeNodeAddresses,
		NodeRandomSeedPublicKeys: nil,
	}

	return res, nil
}

func calculateCommitteeSize(maximumCommitteeSize uint32, minimumCommitteeSize uint32, totalValidatorsSize uint32) uint32 {
	if maximumCommitteeSize < minimumCommitteeSize {
		return minimumCommitteeSize
	}

	if maximumCommitteeSize > totalValidatorsSize {
		return totalValidatorsSize
	}
	return maximumCommitteeSize
}

// See https://github.com/orbs-network/orbs-spec/issues/111
func chooseRandomCommitteeIndices(committeeSize uint32, randomSeed uint64, nodes []primitives.NodeAddress) ([]uint32, error) {

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

	// even if the number of nodes is below minimum, we don't want to crash here and let our caller deal with this
	if uint32(len(nodes)) < committeeSize {
		committeeSize = uint32(len(nodes))
	}

	indices := make([]uint32, committeeSize)
	for i := 0; i < int(committeeSize); i++ {
		indices[i] = grades[i].index
	}
	return indices, nil
}
