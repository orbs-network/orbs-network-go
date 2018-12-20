package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type communication struct {
	logger                  log.BasicLogger
	gossip                  gossiptopics.LeanHelix
	messageReceiversCounter int
	//messageReceivers        map[int]leanhelix.MessageHandler
}

func NewCommunication(logger log.BasicLogger, gossip gossiptopics.LeanHelix) *communication {
	return &communication{
		logger: logger,
		gossip: gossip,
		//messageReceivers:        make(map[int]leanhelix.MessageHandler),
		messageReceiversCounter: 0,
	}
}

// LeanHelix lib sends its messages here
func (comm *communication) SendConsensusMessage(ctx context.Context, lhtargets []lhprimitives.MemberId, consensusRawMessage *leanhelix.ConsensusRawMessage) {
	targets := make([]primitives.NodeAddress, 0, len(lhtargets))
	for _, lhtarget := range lhtargets {
		targets = append(targets, primitives.NodeAddress(lhtarget))
	}

	var blockPair *protocol.BlockPairContainer
	if consensusRawMessage.Block != nil {
		blockPairWrapper := consensusRawMessage.Block.(*BlockPairWrapper)
		if blockPairWrapper != nil {
			blockPair = blockPairWrapper.blockPair
		}
	}

	message := &gossiptopics.LeanHelixInput{
		RecipientsList: &gossiptopics.RecipientsList{
			RecipientNodeAddresses: targets,
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		},
		Message: &gossipmessages.LeanHelixMessage{
			Content:   consensusRawMessage.Content,
			BlockPair: blockPair,
		},
	}
	comm.gossip.SendLeanHelixMessage(ctx, message)
}
