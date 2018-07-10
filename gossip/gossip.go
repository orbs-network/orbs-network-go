package gossip

import (
	"fmt"

	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/messages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	gossip2 "github.com/orbs-network/orbs-spec/types/go/services/gossip"
)

type Config interface {
	NodeId() string
}

type gossip struct {
	transport Transport

	transactionHandlers []gossip2.TransactionRelayHandler
	consensusHandlers   []gossip2.LeanHelixConsensusHandler

	config Config
}

func NewGossip(transport Transport, config Config) services.Gossip {
	g := &gossip{transport: transport, config: config}
	transport.RegisterListener(g, g.config.NodeId())
	return g
}

func (g *gossip) BroadcastForwardedTransactions(input *gossip2.ForwardedTransactionsInput) (*gossip2.TransactionRelayOutput, error) {
	g.transport.Broadcast(&Message{Sender: g.config.NodeId(), Type: ForwardTransactionMessage, Payload: input.Transactions[0].Raw()}) //TODO serialize full input
	return nil, nil
}

func (g *gossip) RegisterTransactionRelayHandler(handler gossip2.TransactionRelayHandler) {
	g.transactionHandlers = append(g.transactionHandlers, handler)
}

func (g *gossip) BroadcastBlockSyncAvailabilityRequest(input *gossip2.BlockSyncAvailabilityRequestInput) (*gossip2.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (g *gossip) SendBlockSyncAvailabilityResponse(input *gossip2.BlockSyncAvailabilityResponseInput) (*gossip2.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (g *gossip) SendBlockSyncRequest(input *gossip2.BlockSyncRequestInput) (*gossip2.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (g *gossip) SendBlockSyncResponse(input *gossip2.BlockSyncResponseInput) (*gossip2.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (g *gossip) RegisterBlockSyncHandler(handler gossip2.BlockSyncHandler) {
	panic("Not implemented")
}

func (g *gossip) SendLeanHelixPrePrepare(input *gossip2.LeanHelixPrePrepareInput) (*gossip2.LeanHelixOutput, error) {
	//TODO write entire input to transport
	return nil, g.transport.Broadcast(&Message{Sender: g.config.NodeId(), Type: PrePrepareMessage, Payload: input.Block})
}

func (g *gossip) SendLeanHelixPrepare(input *gossip2.LeanHelixPrepareInput) (*gossip2.LeanHelixOutput, error) {
	return nil, g.transport.Broadcast(&Message{Sender: g.config.NodeId(), Type: PrepareMessage, Payload: nil})
}

func (g *gossip) SendLeanHelixCommit(input *gossip2.LeanHelixCommitInput) (*gossip2.LeanHelixOutput, error) {
	return nil, g.transport.Broadcast(&Message{Sender: g.config.NodeId(), Type: CommitMessage, Payload: nil})
}

func (g *gossip) SendLeanHelixViewChange(input *gossip2.LeanHelixViewChangeInput) (*gossip2.LeanHelixOutput, error) {
	panic("Not implemented")
}
func (g *gossip) SendLeanHelixNewView(input *gossip2.LeanHelixNewViewInput) (*gossip2.LeanHelixOutput, error) {
	panic("Not implemented")
}
func (g *gossip) RegisterLeanHelixConsensusHandler(handler gossip2.LeanHelixConsensusHandler) {
	g.consensusHandlers = append(g.consensusHandlers, handler)
}

func (g *gossip) OnMessageReceived(message *Message) {
	fmt.Println("Gossip: OnMessageReceived", message)
	fmt.Println("Gossip: Message.Payload", message.Payload)

	switch message.Type {
	case CommitMessage:
		for _, l := range g.consensusHandlers {
			l.HandleLeanHelixCommit(&gossip2.LeanHelixCommitInput{})
		}

	case ForwardTransactionMessage:
		//TODO validate
		tx := protocol.SignedTransactionReader(message.Payload)
		if !tx.IsValid() {
			panic("invalid transaction!")
		}

		for _, l := range g.transactionHandlers {
			l.HandleForwardedTransactions(&gossip2.ForwardedTransactionsInput{Transactions: []*protocol.SignedTransaction{tx}})
		}

	case PrePrepareMessage:
		for _, l := range g.consensusHandlers {
			//l.OnVoteRequest(message.Sender, tx)
			prePrepareMessage := &gossip2.LeanHelixPrePrepareInput{
				Block:  message.Payload,
				Header: (&messages.LeanHelixPrePrepareHeaderBuilder{SenderPublicKey: []byte(message.Sender)}).Build(),
			}
			l.HandleLeanHelixPrePrepare(prePrepareMessage)
		}

	case PrepareMessage:
		for _, l := range g.consensusHandlers {
			l.HandleLeanHelixPrepare(&gossip2.LeanHelixPrepareInput{})
		}
	}
}
