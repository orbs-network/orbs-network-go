package gossip

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type Config interface {
	NodePublicKey() primitives.Ed25519Pkey
}

type service struct {
	transport           adapter.Transport
	transactionHandlers []gossiptopics.TransactionRelayHandler
	consensusHandlers   []gossiptopics.LeanHelixHandler
	config              Config
	reporting           instrumentation.Reporting
}

func NewGossip(transport adapter.Transport, config Config, reporting instrumentation.Reporting) services.Gossip {
	s := &service{
		transport: transport,
		config:    config,
		reporting: reporting,
	}
	transport.RegisterListener(s, s.config.NodePublicKey())
	return s
}

func (s *service) RegisterTransactionRelayHandler(handler gossiptopics.TransactionRelayHandler) {
	s.transactionHandlers = append(s.transactionHandlers, handler)
}

func (s *service) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {
	s.consensusHandlers = append(s.consensusHandlers, handler)
}

func (s *service) RegisterBenchmarkConsensusHandler(handler gossiptopics.BenchmarkConsensusHandler) {
	//s.consensusHandlers = append(s.consensusHandlers, handler)
	panic("Not implemented")
}

func (s *service) BroadcastForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode:    gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic:            gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY,
		TransactionRelay: gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS,
	}).Build()
	payloads := [][]byte{header.Raw(), input.Message.SignedTransactions[0].Raw()}
	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   header.RecipientMode(),
		// TODO: change to input.RecipientList
		Payloads: payloads,
	})
}

func (s *service) BroadcastBlockAvailabilityRequest(input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
func (s *service) SendBlockAvailabilityResponse(input *gossiptopics.BlockAvailabilityResponseInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
func (s *service) SendBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
func (s *service) SendBlockSyncResponse(input *gossiptopics.BlockSyncResponseInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
func (s *service) RegisterBlockSyncHandler(handler gossiptopics.BlockSyncHandler) {
	panic("Not implemented")
}

func (s *service) SendLeanHelixPrePrepare(input *gossiptopics.LeanHelixPrePrepareInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_PRE_PREPARE,
	}).Build()
	payloads := [][]byte{header.Raw(), input.Message.BlockPair.TransactionsBlock.Header.Raw()}
	for _, tx := range input.Message.BlockPair.TransactionsBlock.SignedTransactions {
		payloads = append(payloads, tx.Raw())
	}
	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   header.RecipientMode(),
		// TODO: change to input.RecipientList
		Payloads: payloads,
	})
}

func (s *service) SendLeanHelixPrepare(input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_PREPARE,
	}).Build()
	payloads := [][]byte{header.Raw()}
	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   header.RecipientMode(),
		// TODO: change to input.RecipientList
		Payloads: payloads,
	})
}

func (s *service) SendLeanHelixCommit(input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_COMMIT,
	}).Build()
	payloads := [][]byte{header.Raw()}
	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   header.RecipientMode(),
		// TODO: change to input.RecipientList
		Payloads: payloads,
	})
}

func (s *service) SendLeanHelixViewChange(input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) SendLeanHelixNewView(input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) BroadcastBenchmarkConsensusCommit(input *gossiptopics.BenchmarkConsensusCommitInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) SendBenchmarkConsensusCommitted(input *gossiptopics.BenchmarkConsensusCommittedInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")

}
