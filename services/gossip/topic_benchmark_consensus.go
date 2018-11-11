package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/codec"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) RegisterBenchmarkConsensusHandler(handler gossiptopics.BenchmarkConsensusHandler) {
	s.benchmarkConsensusHandlers = append(s.benchmarkConsensusHandlers, handler)
}

func (s *service) receivedBenchmarkConsensusMessage(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	switch header.BenchmarkConsensus() {
	case consensus.BENCHMARK_CONSENSUS_COMMIT:
		s.receivedBenchmarkConsensusCommit(ctx, header, payloads)
	case consensus.BENCHMARK_CONSENSUS_COMMITTED:
		s.receivedBenchmarkConsensusCommitted(ctx, header, payloads)
	}
}

func (s *service) BroadcastBenchmarkConsensusCommit(ctx context.Context, input *gossiptopics.BenchmarkConsensusCommitInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:              gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: consensus.BENCHMARK_CONSENSUS_COMMIT,
		RecipientMode:      gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	blockPairPayloads, err := codec.EncodeBlockPair(input.Message.BlockPair)
	if err != nil {
		return nil, err
	}
	payloads := append([][]byte{header.Raw()}, blockPairPayloads...)

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:        payloads,
	})
}

func (s *service) receivedBenchmarkConsensusCommit(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	blockPair, err := codec.DecodeBlockPair(payloads)
	if err != nil {
		s.logger.Info("HandleBenchmarkConsensusCommit failed to decode block pair", log.Error(err))
		return
	}

	for _, l := range s.benchmarkConsensusHandlers {
		_, err := l.HandleBenchmarkConsensusCommit(ctx, &gossiptopics.BenchmarkConsensusCommitInput{
			Message: &gossipmessages.BenchmarkConsensusCommitMessage{
				BlockPair: blockPair,
			},
		})
		if err != nil {
			s.logger.Info("HandleBenchmarkConsensusCommit failed", log.Error(err))
		}
	}
}

func (s *service) SendBenchmarkConsensusCommitted(ctx context.Context, input *gossiptopics.BenchmarkConsensusCommittedInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:               gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS,
		BenchmarkConsensus:  consensus.BENCHMARK_CONSENSUS_COMMITTED,
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
	}).Build()

	if input.Message.Status == nil {
		return nil, errors.Errorf("cannot encode BenchmarkConsensusCommittedMessage: %s", input.Message.String())
	}
	payloads := [][]byte{header.Raw(), input.Message.Status.Raw(), input.Message.Sender.Raw()}

	return nil, s.transport.Send(ctx, &adapter.TransportData{
		SenderPublicKey:     s.config.NodePublicKey(),
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519PublicKey{input.RecipientPublicKey},
		Payloads:            payloads,
	})
}

func (s *service) receivedBenchmarkConsensusCommitted(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	if len(payloads) < 2 {
		return
	}
	status := gossipmessages.BenchmarkConsensusStatusReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])

	for _, l := range s.benchmarkConsensusHandlers {
		_, err := l.HandleBenchmarkConsensusCommitted(ctx, &gossiptopics.BenchmarkConsensusCommittedInput{
			Message: &gossipmessages.BenchmarkConsensusCommittedMessage{
				Status: status,
				Sender: senderSignature,
			},
		})
		if err != nil {
			s.logger.Info("HandleBenchmarkConsensusCommitted failed", log.Error(err))
		}
	}
}
