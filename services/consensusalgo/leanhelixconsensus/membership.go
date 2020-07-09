// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"context"
	"fmt"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"strconv"
	"strings"
)

type membership struct {
	memberId         primitives.NodeAddress
	consensusContext services.ConsensusContext
	logger           log.Logger
	maxCommitteeSize uint32
}

func NewMembership(logger log.Logger, memberId primitives.NodeAddress, consensusContext services.ConsensusContext, maxCommitteeSize uint32) *membership {
	if consensusContext == nil {
		panic("consensusContext cannot be nil")
	}
	logger.Info("NewMembership()", log.Stringable("ID", memberId))
	return &membership{
		consensusContext: consensusContext,
		logger:           logger,
		memberId:         memberId,
		maxCommitteeSize: maxCommitteeSize,
	}
}

func (m *membership) MyMemberId() lhprimitives.MemberId {
	return lhprimitives.MemberId(m.memberId)
}

func (m *membership) RequestOrderedCommittee(ctx context.Context, blockHeight lhprimitives.BlockHeight, seed uint64, prevBlockReferenceTime lhprimitives.TimestampSeconds) ([]lh.CommitteeMember, error) {
	res, err := m.consensusContext.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
		CurrentBlockHeight: primitives.BlockHeight(blockHeight),
		RandomSeed:         seed,
		MaxCommitteeSize:   m.maxCommitteeSize,
		PrevBlockReferenceTime: primitives.TimestampSeconds(prevBlockReferenceTime),
	})
	if err != nil {
		m.logger.Info(" failed RequestOrderedCommittee()", log.Error(err))
		return nil, err
	}

	committeeMembers := toMembers(res.NodeAddresses, res.Weights)
	committeeMembersStr := toMembersString(res.NodeAddresses, res.Weights)
	// random-seed printed as string for logz.io, do not change it back to log.Uint64()
	m.logger.Info("Received committee members", logfields.BlockHeight(primitives.BlockHeight(blockHeight)), log.Uint32("prev-block-ref-time", uint32(prevBlockReferenceTime)), log.String("random-seed", strconv.FormatUint(seed, 10)), log.String("committee-members", committeeMembersStr))

	return committeeMembers, nil
}

func (m *membership) RequestCommitteeForBlockProof(ctx context.Context, prevBlockReferenceTime lhprimitives.TimestampSeconds) ([]lh.CommitteeMember, error) {
	res, err := m.consensusContext.RequestBlockProofOrderingCommittee(ctx, &services.RequestBlockProofCommitteeInput{
		PrevBlockReferenceTime: primitives.TimestampSeconds(prevBlockReferenceTime),
	})
	if err != nil {
		m.logger.Info(" failed RequestCommitteeForBlockProof()", log.Error(err))
		return nil, err
	}

	committeeMembers := toMembers(res.NodeAddresses, res.Weights)
	committeeMembersStr := toMembersString(res.NodeAddresses, res.Weights)
	m.logger.Info("Received committee members for block proof", log.Uint32("prev-block-ref-time", uint32(prevBlockReferenceTime)), log.String("committee-members", committeeMembersStr))

	return committeeMembers, nil
}

func toMembers(nodeAddresses []primitives.NodeAddress, weights []primitives.Weight) []lh.CommitteeMember {
	members := make([]lh.CommitteeMember, len(nodeAddresses))
	for i := range nodeAddresses {
		members[i].Id = lhprimitives.MemberId(nodeAddresses[i])
		members[i].Weight = lhprimitives.MemberWeight(weights[i])
	}
	return members
}

func toMembersString(nodeAddresses []primitives.NodeAddress, weights []primitives.Weight) string {
	members := make([]string, len(nodeAddresses))
	for i := range nodeAddresses {
		members[i] = fmt.Sprintf("{\"Address:\": \"%v\", \"Weight\": %d}", nodeAddresses[i], weights[i])  // %v is because NodeAddress has .String()
	}
	return strings.Join(members, ",")
}


