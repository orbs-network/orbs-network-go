package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) RegisterLeanHelixHandler(handler gossiptopics.LeanHelixHandler) {
	s.leanHelixHandlers = append(s.leanHelixHandlers, handler)
}

func (s *service) receivedLeanHelixMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	switch header.LeanHelix() {
	case consensus.LEAN_HELIX_PRE_PREPARE:
		s.receivedLeanHelixPrePrepare(ctx, header, payloads)
	case consensus.LEAN_HELIX_PREPARE:
		s.receivedLeanHelixPrepare(ctx, header, payloads)
	case consensus.LEAN_HELIX_COMMIT:
		s.receivedLeanHelixCommit(ctx, header, payloads)
	}
}

func (s *service) SendLeanHelixPrePrepare(ctx context.Context, input *gossiptopics.LeanHelixPrePrepareInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_PRE_PREPARE,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	blockPairPayloads, err := encodeBlockPair(input.Message.BlockPair)
	if err != nil {
		return nil, err
	}
	if input.Message.SignedHeader == nil || input.Message.Sender == nil {
		return nil, errors.Errorf("cannot encode LeanHelixPrePrepareMessage: %s", input.Message.String())
	}
	payloads := append([][]byte{header.Raw(), input.Message.SignedHeader.Raw(), input.Message.Sender.Raw()}, blockPairPayloads...)

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

func (s *service) receivedLeanHelixPrePrepare(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	signedHeader := consensus.LeanHelixBlockRefReader(payloads[0])
	senderSignature := consensus.LeanHelixSenderSignatureReader(payloads[1])
	blockPair, err := decodeBlockPair(payloads[2:])
	if err != nil {
		return
	}

	for _, l := range s.leanHelixHandlers {
		_, err := l.HandleLeanHelixPrePrepare(ctx, &gossiptopics.LeanHelixPrePrepareInput{
			Message: &gossipmessages.LeanHelixPrePrepareMessage{
				SignedHeader: signedHeader,
				Sender:       senderSignature,
				BlockPair:    blockPair,
			},
		})
		if err != nil {
			s.logger.Info("HandleLeanHelixPrePrepare failed", log.Error(err))
		}
	}
}

func (s *service) SendLeanHelixPrepare(ctx context.Context, input *gossiptopics.LeanHelixPrepareInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_PREPARE,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	payloads := [][]byte{header.Raw()}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

func (s *service) receivedLeanHelixPrepare(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	for _, l := range s.leanHelixHandlers {
		_, err := l.HandleLeanHelixPrepare(ctx, &gossiptopics.LeanHelixPrepareInput{})
		if err != nil {
			s.logger.Info("HandleLeanHelixPrepare failed", log.Error(err))
		}
	}
}

func (s *service) SendLeanHelixCommit(ctx context.Context, input *gossiptopics.LeanHelixCommitInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		LeanHelix:     consensus.LEAN_HELIX_COMMIT,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	payloads := [][]byte{header.Raw()}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST, // TODO: shouldn't be broadcast
		Payloads:        payloads,
	})
}

func (s *service) receivedLeanHelixCommit(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	for _, l := range s.leanHelixHandlers {
		_, err := l.HandleLeanHelixCommit(ctx, &gossiptopics.LeanHelixCommitInput{})
		if err != nil {
			s.logger.Info("HandleLeanHelixCommit failed", log.Error(err))
		}
	}
}

func (s *service) SendLeanHelixViewChange(ctx context.Context, input *gossiptopics.LeanHelixViewChangeInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}

func (s *service) SendLeanHelixNewView(ctx context.Context, input *gossiptopics.LeanHelixNewViewInput) (*gossiptopics.EmptyOutput, error) {
	panic("Not implemented")
}
