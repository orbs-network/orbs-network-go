// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

func (s *service) RequestOrderingCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	return s.RequestValidationCommittee(ctx, input)
}

func (s *service) RequestValidationCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))
	orderedCommittee, err := s.getOrderedCommittee(ctx, input.CurrentBlockHeight)
	if err != nil {
		return nil, err
	}

	committeeSize := calculateCommitteeSize(input.MaxCommitteeSize, s.config.LeanHelixConsensusMinimumCommitteeSize(), uint32(len(orderedCommittee)))
	logger.Info("Calculated committee size", logfields.BlockHeight(input.CurrentBlockHeight), log.Uint32("committee-size", committeeSize), log.Int("elected-validators-count", len(orderedCommittee)), log.Uint32("max-committee-size", input.MaxCommitteeSize))

	res := &services.RequestCommitteeOutput{
		NodeAddresses:            orderedCommittee[:committeeSize],
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
