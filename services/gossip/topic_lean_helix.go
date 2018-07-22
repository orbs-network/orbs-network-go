package gossip

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {
	s.leanHelixHandlers = append(s.leanHelixHandlers, handler)
}

func (s *service) receivedLeanHelixMessage(header *gossipmessages.Header, payloads [][]byte) {
	switch header.LeanHelix() {
	case consensus.LEAN_HELIX_PRE_PREPARE:
		s.receivedLeanHelixPrePrepare(header, payloads)
	case consensus.LEAN_HELIX_PREPARE:
		s.receivedLeanHelixPrepare(header, payloads)
	case consensus.LEAN_HELIX_COMMIT:
		s.receivedLeanHelixCommit(header, payloads)
	}
}

func (s *service) SendLeanHelixPrePrepare(input *gossiptopics.LeanHelixPrePrepareInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_PRE_PREPARE,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	payloads := [][]byte{
		header.Raw(),
		input.Message.BlockPair.TransactionsBlock.Header.Raw(),
	}
	for _, tx := range input.Message.BlockPair.TransactionsBlock.SignedTransactions {
		payloads = append(payloads, tx.Raw())
	}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

func (s *service) receivedLeanHelixPrePrepare(header *gossipmessages.Header, payloads [][]byte) {
	defer func() { recover() }() // this will make sure we don't crash on out of bounds on byzantine messages
	txBlockHeader := protocol.TransactionsBlockHeaderReader(payloads[0])
	txs := []*protocol.SignedTransaction{}
	for _, payload := range payloads[1:] {
		txs = append(txs, protocol.SignedTransactionReader(payload))
	}

	for _, l := range s.leanHelixHandlers {
		l.HandleLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
			Message: &gossipmessages.LeanHelixPrePrepareMessage{
				BlockPair: &protocol.BlockPairContainer{
					TransactionsBlock: &protocol.TransactionsBlockContainer{
						Header:             txBlockHeader,
						SignedTransactions: txs,
					},
				},
			},
		})
	}
}

func (s *service) SendLeanHelixPrepare(input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_PREPARE,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	payloads := [][]byte{header.Raw()}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

func (s *service) receivedLeanHelixPrepare(header *gossipmessages.Header, payloads [][]byte) {
	for _, l := range s.leanHelixHandlers {
		l.HandleLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
	}
}

func (s *service) SendLeanHelixCommit(input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_COMMIT,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	payloads := [][]byte{header.Raw()}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

func (s *service) receivedLeanHelixCommit(header *gossipmessages.Header, payloads [][]byte) {
	for _, l := range s.leanHelixHandlers {
		l.HandleLeanHelixCommit(&gossiptopics.LeanHelixCommitInput{})
	}
}

func (s *service) SendLeanHelixViewChange(input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) SendLeanHelixNewView(input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
