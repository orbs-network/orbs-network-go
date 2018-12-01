package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type networkCommunication struct {
	consensusContext        services.ConsensusContext
	logger                  log.BasicLogger
	gossip                  gossiptopics.LeanHelix
	messageReceiversCounter int
	//messageReceivers        map[int]leanhelix.MessageHandler
}

func NewNetworkCommunication(logger log.BasicLogger, consensusContext services.ConsensusContext, gossip gossiptopics.LeanHelix) *networkCommunication {
	if consensusContext == nil {
		panic("consensusContext cannot be nil")
	}
	return &networkCommunication{
		consensusContext: consensusContext,
		logger:           logger,
		gossip:           gossip,
		//messageReceivers:        make(map[int]leanhelix.MessageHandler),
		messageReceiversCounter: 0,
	}
}

func (comm *networkCommunication) RequestOrderedCommittee(ctx context.Context, blockHeight lhprimitives.BlockHeight, seed uint64, maxCommitteeSize uint32) []lhprimitives.Ed25519PublicKey {

	res, err := comm.consensusContext.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
		BlockHeight:      primitives.BlockHeight(blockHeight),
		RandomSeed:       seed,
		MaxCommitteeSize: maxCommitteeSize,
	})
	if err != nil {
		comm.logger.Info(" failed RequestOrderedCommittee()", log.Error(err))
		return nil
	}
	publicKeys := make([]lhprimitives.Ed25519PublicKey, 0, len(res.NodePublicKeys))
	for _, publicKey := range res.NodePublicKeys {
		publicKeys = append(publicKeys, lhprimitives.Ed25519PublicKey(publicKey))
	}

	return publicKeys
}

func (comm *networkCommunication) IsMember(pk lhprimitives.Ed25519PublicKey) bool {
	panic("implement me")
}

// LeanHelix lib sends its messages here
func (comm *networkCommunication) SendMessage(ctx context.Context, lhtargets []lhprimitives.Ed25519PublicKey, consensusRawMessage leanhelix.ConsensusRawMessage) {
	targets := make([]primitives.Ed25519PublicKey, 0, len(lhtargets))
	for _, lhtarget := range lhtargets {
		targets = append(targets, primitives.Ed25519PublicKey(lhtarget))
	}

	var blockPair *protocol.BlockPairContainer
	if consensusRawMessage.Block() != nil {
		blockPairWrapper := consensusRawMessage.Block().(*BlockPairWrapper)
		if blockPairWrapper != nil {
			blockPair = blockPairWrapper.blockPair
		}
	}

	comm.logger.Info("leanhelix.comm.SendMessage()", log.Stringable("message-type", consensusRawMessage.MessageType()), log.Stringable("block-pair", blockPair))
	message := &gossiptopics.LeanHelixInput{
		RecipientsList: &gossiptopics.RecipientsList{
			RecipientPublicKeys: targets,
			RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		},
		Message: &gossipmessages.LeanHelixMessage{
			MessageType: consensus.LeanHelixMessageType(consensusRawMessage.MessageType()),
			Content:     consensusRawMessage.Content(),
			BlockPair:   blockPair,
		},
	}
	comm.gossip.SendLeanHelixMessage(ctx, message)
}
