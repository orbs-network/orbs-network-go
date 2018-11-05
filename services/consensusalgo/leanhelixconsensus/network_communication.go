package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {

	message := leanhelix.CreateConsensusRawMessage(
		leanhelix.MessageType(input.Message.MessageType),
		input.Message.Content,
		&BlockPairWrapper{
			blockPair: input.Message.BlockPair,
		},
	)

	for _, messageReceiver := range s.messageReceivers {
		messageReceiver(ctx, message)
	}
	return nil, nil
}

func (s *service) RequestOrderedCommittee(seed uint64) []lhprimitives.Ed25519PublicKey {
	panic("implement me")
}

func (s *service) IsMember(pk lhprimitives.Ed25519PublicKey) bool {
	panic("implement me")
}

// Lib calls this method to register itself for incoming messages, and supplies the callback
func (s *service) RegisterOnMessage(onReceivedMessage func(ctx context.Context, message leanhelix.ConsensusRawMessage)) int {
	s.messageReceiversCounter++
	s.messageReceivers[s.messageReceiversCounter] = onReceivedMessage

	return s.messageReceiversCounter
}

func (s *service) UnregisterOnMessage(subscriptionToken int) {
	delete(s.messageReceivers, subscriptionToken)
}

// LeanHelix lib sends its messages here
func (s *service) SendMessage(ctx context.Context, lhtargets []lhprimitives.Ed25519PublicKey, consensusRawMessage leanhelix.ConsensusRawMessage) {

	targets := make([]primitives.Ed25519PublicKey, 0, len(lhtargets))
	for i, lhtarget := range lhtargets {
		targets[i] = primitives.Ed25519PublicKey(lhtarget)
	}

	blockPairWrapper := consensusRawMessage.Block().(*BlockPairWrapper)

	message := &gossiptopics.LeanHelixInput{
		RecipientsList: &gossiptopics.RecipientsList{
			RecipientPublicKeys: targets,
			RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		},
		Message: &gossipmessages.LeanHelixMessage{
			MessageType: consensus.LeanHelixMessageType(consensusRawMessage.MessageType()),
			Content:     consensusRawMessage.Content(),
			BlockPair:   blockPairWrapper.blockPair,
		},
	}
	s.gossip.SendLeanHelixMessage(ctx, message)
}
