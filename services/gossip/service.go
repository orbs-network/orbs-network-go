package gossip

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type Config interface {
	NodeId() string
}

type service struct {
	services.Gossip
	transport           adapter.Transport
	transactionHandlers []gossiptopics.TransactionRelayHandler
	consensusHandlers   []gossiptopics.LeanHelixHandler
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

func (s *service) RegisterTransactionRelayHandler(handler gossiptopics.TransactionRelayHandler) {
	s.transactionHandlers = append(s.transactionHandlers, handler)
}

func (s *service) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {
	s.consensusHandlers = append(s.consensusHandlers, handler)
}

func (s *service) BroadcastForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.TransactionRelayOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic: gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY,
		TransactionRelay: gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS,
		NumPayloads: 1,
	}).Build()
	return nil, s.transport.Send(header, [][]byte{input.Transactions[0].Raw()})
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
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic: gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix: gossipmessages.LEAN_HELIX_PRE_PREPARE,
		NumPayloads: 1,
	}).Build()
	return nil, s.transport.Send(header, [][]byte{input.Block})
}

func (s *service) SendLeanHelixPrepare(input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.LeanHelixOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic: gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix: gossipmessages.LEAN_HELIX_PREPARE,
		NumPayloads: 0,
	}).Build()
	return nil, s.transport.Send(header, [][]byte{})
}

func (s *service) SendLeanHelixCommit(input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.LeanHelixOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic: gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix: gossipmessages.LEAN_HELIX_COMMIT,
		NumPayloads: 0,
	}).Build()
	return nil, s.transport.Send(header, [][]byte{})
}

func (s *service) SendLeanHelixViewChange(input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.LeanHelixOutput, error) {
	panic("Not implemented")
}

func (s *service) SendLeanHelixNewView(input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.LeanHelixOutput, error) {
	panic("Not implemented")
}
