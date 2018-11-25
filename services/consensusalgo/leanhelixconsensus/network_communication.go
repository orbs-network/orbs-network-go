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

type networkCommunication struct {
	gossip                  gossiptopics.LeanHelix
	messageReceiversCounter int
	messageReceivers        map[int]func(ctx context.Context, message leanhelix.ConsensusRawMessage)
}

func NewNetworkCommunication(gossip gossiptopics.LeanHelix) *networkCommunication {
	return &networkCommunication{
		gossip:                  gossip,
		messageReceivers:        make(map[int]func(ctx context.Context, message leanhelix.ConsensusRawMessage)),
		messageReceiversCounter: 0,
	}
}

func (comm *networkCommunication) RequestOrderedCommittee(seed uint64) []lhprimitives.Ed25519PublicKey {
	panic("implement me")
}

func (comm *networkCommunication) IsMember(pk lhprimitives.Ed25519PublicKey) bool {
	panic("implement me")
}

// Lib calls this method to register itself for incoming messages, and supplies the callback
func (comm *networkCommunication) RegisterOnMessage(onReceivedMessage func(ctx context.Context, message leanhelix.ConsensusRawMessage)) int {
	comm.messageReceiversCounter++
	comm.messageReceivers[comm.messageReceiversCounter] = onReceivedMessage

	return comm.messageReceiversCounter
}

func (comm *networkCommunication) UnregisterOnMessage(subscriptionToken int) {
	delete(comm.messageReceivers, subscriptionToken)
}

// LeanHelix lib sends its messages here
func (comm *networkCommunication) SendMessage(ctx context.Context, lhtargets []lhprimitives.Ed25519PublicKey, consensusRawMessage leanhelix.ConsensusRawMessage) {
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
	comm.gossip.SendLeanHelixMessage(ctx, message)
}

func (comm *networkCommunication) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	message := leanhelix.CreateConsensusRawMessage(
		leanhelix.MessageType(input.Message.MessageType),
		input.Message.Content,
		&BlockPairWrapper{
			blockPair: input.Message.BlockPair,
		},
	)

	for _, messageReceiver := range comm.messageReceivers {
		messageReceiver(ctx, message)
	}
	return nil, nil
}

func (comm *networkCommunication) CountRegisteredOnMessage() int {
	return len(comm.messageReceivers)
}
