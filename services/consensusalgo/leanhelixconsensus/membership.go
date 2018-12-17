package leanhelixconsensus

import (
	"context"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type membership struct {
	consensusContext services.ConsensusContext
	logger           log.BasicLogger
	memberId         primitives.Ed25519PublicKey
}

func (m *membership) MyMemberId() lhprimitives.MemberId {
	return lhprimitives.MemberId(m.memberId)
}

func NewMembership(logger log.BasicLogger, memberId primitives.Ed25519PublicKey, consensusContext services.ConsensusContext) *membership {
	if consensusContext == nil {
		panic("consensusContext cannot be nil")
	}
	return &membership{
		consensusContext: consensusContext,
		logger:           logger,
		memberId:         memberId,
	}
}

func (m *membership) RequestOrderedCommittee(ctx context.Context, blockHeight lhprimitives.BlockHeight, seed uint64, maxCommitteeSize uint32) []lhprimitives.MemberId {

	res, err := m.consensusContext.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
		BlockHeight:      primitives.BlockHeight(blockHeight),
		RandomSeed:       seed,
		MaxCommitteeSize: maxCommitteeSize,
	})
	if err != nil {
		m.logger.Info(" failed RequestOrderedCommittee()", log.Error(err))
		return nil
	}
	publicKeys := make([]lhprimitives.MemberId, 0, len(res.NodePublicKeys))
	for _, publicKey := range res.NodePublicKeys {
		publicKeys = append(publicKeys, lhprimitives.MemberId(publicKey))
	}

	return publicKeys
}
