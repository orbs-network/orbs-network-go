// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"strings"
)

func (s *service) RequestOrderingCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	return s.RequestValidationCommittee(ctx, input)
}

func (s *service) RequestValidationCommittee(ctx context.Context, input *services.RequestCommitteeInput) (*services.RequestCommitteeOutput, error) {
	committee, err := s.generateCommitteeUsingContract(ctx, input)
	if err != nil {
		return nil, err
	}

	s.metrics.committeeSize.Update(int64(len(committee)))
	committeeStr := make([]string, len(committee))
	for i, nodeAddress := range committee {
		committeeStr[i] = lhprimitives.MemberId(nodeAddress).String()
	}
	s.metrics.committeeMembers.Update(strings.Join(committeeStr, ","))
	res := &services.RequestCommitteeOutput{
		NodeAddresses:            committee,
		NodeRandomSeedPublicKeys: nil,
	}
	return res, nil
}
