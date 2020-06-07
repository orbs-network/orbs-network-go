// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"strings"
)

func (s *service) RequestOrderingCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	return s.RequestValidationCommittee(ctx, input)
}

func (s *service) RequestValidationCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	committee, err := s.getOrderedCommittee(ctx, input.CurrentBlockHeight, input.PrevBlockReferenceTime)
	if err != nil {
		return nil, err
	}

	s.logger.Info("committee size", logfields.BlockHeight(input.CurrentBlockHeight), log.Int("elected-validators-count", len(committee)), log.Uint32("max-committee-size", input.MaxCommitteeSize), trace.LogFieldFrom(ctx))

	s.metrics.committeeSize.Update(int64(len(committee)))
	committeeStringArray := make([]string, len(committee))
	for j, nodeAddress := range committee {
		committeeStringArray[j] = fmt.Sprintf("\"%v\"", nodeAddress)  // %v is because NodeAddress has .String()
	}
	s.metrics.committeeMembers.Update("[" + strings.Join(committeeStringArray, ", ") + "]")
	s.metrics.committeeRefTime.Update(int64(input.PrevBlockReferenceTime))

	res := &services.RequestCommitteeOutput{
		NodeAddresses:            committee,
		NodeRandomSeedPublicKeys: nil,
	}
	return res, nil
}


func (s *service) RequestBlockProofOrderingCommittee(ctx context.Context, input *services.RequestBlockProofCommitteeInput) (*services.RequestBlockProofCommitteeOutput, error) {
	return s.RequestBlockProofValidationCommittee(ctx, input)
}

func (s *service) RequestBlockProofValidationCommittee(ctx context.Context, input *services.RequestBlockProofCommitteeInput) (*services.RequestBlockProofCommitteeOutput, error) {
	out, err := s.management.GetCommittee(ctx, &services.GetCommitteeInput{Reference:input.PrevBlockReferenceTime})
	if err != nil {
		return nil, err
	}
	committee := out.Members
	s.logger.Info("committee size", log.Int("elected-validators-count", len(committee)), trace.LogFieldFrom(ctx))

	committeeStringArray := make([]string, len(committee))
	for j, nodeAddress := range committee {
		committeeStringArray[j] = fmt.Sprintf("\"%v\"", nodeAddress)  // %v is because NodeAddress has .String()
	}

	res := &services.RequestBlockProofCommitteeOutput{
		NodeAddresses:            committee,
	}
	return res, nil
}