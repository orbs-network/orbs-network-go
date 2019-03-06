package leanhelixconsensus

import (
	"context"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type membership struct {
	memberId         primitives.NodeAddress
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	committeeSize    uint32
}

func NewMembership(logger log.BasicLogger, memberId primitives.NodeAddress, consensusContext services.ConsensusContext, committeeSize uint32) *membership {
	if consensusContext == nil {
		panic("consensusContext cannot be nil")
	}
	logger.Info("NewMembership()", log.Stringable("ID", memberId))
	return &membership{
		consensusContext: consensusContext,
		logger:           logger,
		memberId:         memberId,
		committeeSize:    committeeSize,
	}
}
func (m *membership) MyMemberId() lhprimitives.MemberId {
	return lhprimitives.MemberId(m.memberId)
}

func (m *membership) RequestOrderedCommittee(ctx context.Context, blockHeight lhprimitives.BlockHeight, seed uint64) ([]lhprimitives.MemberId, error) {

	res, err := m.consensusContext.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
		CurrentBlockHeight: primitives.BlockHeight(blockHeight),
		RandomSeed:         seed,
		MaxCommitteeSize:   m.committeeSize,
	})
	if err != nil {
		m.logger.Info(" failed RequestOrderedCommittee()", log.Error(err))
		return nil, err
	}

	nodeAddresses := toMemberIds(res.NodeAddresses)

	return nodeAddresses, nil
}

func toMemberIds(nodeAddresses []primitives.NodeAddress) []lhprimitives.MemberId {
	memberIds := make([]lhprimitives.MemberId, 0, len(nodeAddresses))
	for _, nodeAddress := range nodeAddresses {
		memberIds = append(memberIds, lhprimitives.MemberId(nodeAddress))
	}
	return memberIds
}
