// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"context"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"strconv"
	"strings"
)

type membership struct {
	memberId         primitives.NodeAddress
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	maxCommitteeSize uint32
}

func NewMembership(logger log.BasicLogger, memberId primitives.NodeAddress, consensusContext services.ConsensusContext, maxCommitteeSize uint32) *membership {
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

func nodeAddressesToCommaSeparatedString(nodeAddresses []primitives.NodeAddress) string {
	addrs := make([]string, 0)
	for _, nodeAddress := range nodeAddresses {
		addrs = append(addrs, nodeAddress.String())
	}
	return strings.Join(addrs, ",")
}

func (m *membership) RequestOrderedCommittee(ctx context.Context, blockHeight lhprimitives.BlockHeight, seed uint64) ([]lhprimitives.MemberId, error) {
	res, err := m.consensusContext.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
		CurrentBlockHeight: primitives.BlockHeight(blockHeight),
		RandomSeed:         seed,
		MaxCommitteeSize:   m.maxCommitteeSize,
	})
	if err != nil {
		m.logger.Info(" failed RequestOrderedCommittee()", log.Error(err))
		return nil, err
	}

	nodeAddresses := toMemberIds(res.NodeAddresses)
	committeeMembersStr := nodeAddressesToCommaSeparatedString(res.NodeAddresses)
	// random-seed printed as string for logz.io, do not change it back to log.Uint64()
	m.logger.Info("Received committee members", log.BlockHeight(primitives.BlockHeight(blockHeight)), log.String("random-seed", strconv.FormatUint(seed, 10)), log.String("committee-members", committeeMembersStr))

	return nodeAddresses, nil
}

func toMemberIds(nodeAddresses []primitives.NodeAddress) []lhprimitives.MemberId {
	memberIds := make([]lhprimitives.MemberId, 0, len(nodeAddresses))
	for _, nodeAddress := range nodeAddresses {
		memberIds = append(memberIds, lhprimitives.MemberId(nodeAddress))
	}
	return memberIds
}
