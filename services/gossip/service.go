package gossip

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
)

type Config interface {
	NodeId() string
}

type service struct {
	services.Gossip
	transport           adapter.Transport
	transactionHandlers []gossiptopics.TransactionRelayHandler
	consensusHandlers   []gossiptopics.LeanHelixConsensusHandler
	config              Config
}

func NewGossip(transport adapter.Transport, config Config) services.Gossip {
	s := &service{
		transport: transport,
		config:    config,
	}
	transport.RegisterListener(s, s.config.NodeId())
	return s
}

func (s *service) BroadcastForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.TransactionRelayOutput, error) {
	s.transport.Broadcast(&adapter.Message{Sender: s.config.NodeId(), Type: adapter.ForwardTransactionMessage, Payload: input.Transactions[0].Raw()}) //TODO serialize full input
	return nil, nil
}

func (s *service) RegisterTransactionRelayHandler(handler gossiptopics.TransactionRelayHandler) {
	s.transactionHandlers = append(s.transactionHandlers, handler)
}

func (s *service) BroadcastBlockSyncAvailabilityRequest(input *gossiptopics.BlockSyncAvailabilityRequestInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) SendBlockSyncAvailabilityResponse(input *gossiptopics.BlockSyncAvailabilityResponseInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) SendBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) SendBlockSyncResponse(input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.BlockSyncOutput, error) {
	panic("Not implemented")
}
func (s *service) RegisterBlockSyncHandler(handler gossiptopics.BlockSyncHandler) {
	panic("Not implemented")
}

func (s *service) SendLeanHelixPrePrepare(input *gossiptopics.LeanHelixPrePrepareInput) (*gossiptopics.LeanHelixOutput, error) {
	//TODO write entire input to transport
	return nil, s.transport.Broadcast(&adapter.Message{Sender: s.config.NodeId(), Type: adapter.PrePrepareMessage, Payload: input.Block})
}

func (s *service) SendLeanHelixPrepare(input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.LeanHelixOutput, error) {
	return nil, s.transport.Broadcast(&adapter.Message{Sender: s.config.NodeId(), Type: adapter.PrepareMessage, Payload: nil})
}

func (s *service) SendLeanHelixCommit(input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.LeanHelixOutput, error) {
	return nil, s.transport.Broadcast(&adapter.Message{Sender: s.config.NodeId(), Type: adapter.CommitMessage, Payload: nil})
}

func (s *service) SendLeanHelixViewChange(input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.LeanHelixOutput, error) {
	panic("Not implemented")
}
func (s *service) SendLeanHelixNewView(input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.LeanHelixOutput, error) {
	panic("Not implemented")
}
func (s *service) RegisterLeanHelixConsensusHandler(handler gossiptopics.LeanHelixConsensusHandler) {
	s.consensusHandlers = append(s.consensusHandlers, handler)
}