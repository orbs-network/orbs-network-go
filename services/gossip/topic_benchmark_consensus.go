package gossip

import (
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) RegisterBenchmarkConsensusHandler(handler gossiptopics.BenchmarkConsensusHandler) {
	s.benchmarkConsensusHandlers = append(s.benchmarkConsensusHandlers, handler)
}

func (s *service) receivedBenchmarkConsensusMessage(header *gossipmessages.Header, payloads [][]byte) {
	switch header.BenchmarkConsensus() {
	case consensus.BENCHMARK_CONSENSUS_COMMIT:
		s.receivedBenchmarkConsensusCommit(header, payloads)
	case consensus.BENCHMARK_CONSENSUS_COMMITTED:
		s.receivedBenchmarkConsensusCommitted(header, payloads)
	}
}

func (s *service) BroadcastBenchmarkConsensusCommit(input *gossiptopics.BenchmarkConsensusCommitInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:              gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: consensus.BENCHMARK_CONSENSUS_COMMIT,
		RecipientMode:      gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()

	payloads := make([][]byte, 0, 6+
		len(input.Message.BlockPair.TransactionsBlock.SignedTransactions)+
		len(input.Message.BlockPair.ResultsBlock.TransactionReceipts)+
		len(input.Message.BlockPair.ResultsBlock.ContractStateDiffs),
	)
	payloads = append(payloads, header.Raw())
	payloads = append(payloads, input.Message.BlockPair.TransactionsBlock.Header.Raw())
	payloads = append(payloads, input.Message.BlockPair.TransactionsBlock.Metadata.Raw())
	payloads = append(payloads, input.Message.BlockPair.TransactionsBlock.BlockProof.Raw())
	payloads = append(payloads, input.Message.BlockPair.ResultsBlock.Header.Raw())
	payloads = append(payloads, input.Message.BlockPair.ResultsBlock.BlockProof.Raw())
	for _, tx := range input.Message.BlockPair.TransactionsBlock.SignedTransactions {
		payloads = append(payloads, tx.Raw())
	}
	for _, receipt := range input.Message.BlockPair.ResultsBlock.TransactionReceipts {
		payloads = append(payloads, receipt.Raw())
	}
	for _, sdiff := range input.Message.BlockPair.ResultsBlock.ContractStateDiffs {
		payloads = append(payloads, sdiff.Raw())
	}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey: s.config.NodePublicKey(),
		RecipientMode:   gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		Payloads:        payloads,
	})
}

func (s *service) receivedBenchmarkConsensusCommit(header *gossipmessages.Header, payloads [][]byte) {
	defer func() { recover() }() // this will make sure we don't crash on out of bounds on byzantine messages
	txBlockHeader := protocol.TransactionsBlockHeaderReader(payloads[0])
	txBlockMetadata := protocol.TransactionsBlockMetadataReader(payloads[1])
	txBlockProof := protocol.TransactionsBlockProofReader(payloads[2])
	rxBlockHeader := protocol.ResultsBlockHeaderReader(payloads[3])
	rxBlockProof := protocol.ResultsBlockProofReader(payloads[4])
	payloadIndex := uint32(5)
	txs := make([]*protocol.SignedTransaction, 0, txBlockHeader.NumSignedTransactions())
	for i := uint32(0); i < txBlockHeader.NumSignedTransactions(); i++ {
		txs = append(txs, protocol.SignedTransactionReader(payloads[payloadIndex+i]))
	}
	payloadIndex += txBlockHeader.NumSignedTransactions()
	receipts := make([]*protocol.TransactionReceipt, 0, rxBlockHeader.NumTransactionReceipts())
	for i := uint32(0); i < rxBlockHeader.NumTransactionReceipts(); i++ {
		receipts = append(receipts, protocol.TransactionReceiptReader(payloads[payloadIndex+i]))
	}
	payloadIndex += rxBlockHeader.NumTransactionReceipts()
	sdiffs := make([]*protocol.ContractStateDiff, 0, rxBlockHeader.NumContractStateDiffs())
	for i := uint32(0); i < rxBlockHeader.NumContractStateDiffs(); i++ {
		sdiffs = append(sdiffs, protocol.ContractStateDiffReader(payloads[payloadIndex+i]))
	}

	for _, l := range s.benchmarkConsensusHandlers {
		l.HandleBenchmarkConsensusCommit(&gossiptopics.BenchmarkConsensusCommitInput{
			Message: &gossipmessages.BenchmarkConsensusCommitMessage{
				BlockPair: &protocol.BlockPairContainer{
					TransactionsBlock: &protocol.TransactionsBlockContainer{
						Header:             txBlockHeader,
						Metadata:           txBlockMetadata,
						SignedTransactions: txs,
						BlockProof:         txBlockProof,
					},
					ResultsBlock: &protocol.ResultsBlockContainer{
						Header:              rxBlockHeader,
						TransactionReceipts: receipts,
						ContractStateDiffs:  sdiffs,
						BlockProof:          rxBlockProof,
					},
				},
			},
		})
	}
}

func (s *service) SendBenchmarkConsensusCommitted(input *gossiptopics.BenchmarkConsensusCommittedInput) (*gossiptopics.EmptyOutput, error) {
	header := (&gossipmessages.HeaderBuilder{
		Topic:               gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS,
		BenchmarkConsensus:  consensus.BENCHMARK_CONSENSUS_COMMITTED,
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519Pkey{input.RecipientPublicKey},
	}).Build()
	senderSignature := (&gossipmessages.SenderSignatureBuilder{
		SenderPublicKey: s.config.NodePublicKey(),
	}).Build()

	payloads := [][]byte{
		header.Raw(),
		input.Message.Status.Raw(),
		senderSignature.Raw(),
	}

	return nil, s.transport.Send(&adapter.TransportData{
		SenderPublicKey:     s.config.NodePublicKey(),
		RecipientMode:       gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientPublicKeys: []primitives.Ed25519Pkey{input.RecipientPublicKey},
		Payloads:            payloads,
	})
}

func (s *service) receivedBenchmarkConsensusCommitted(header *gossipmessages.Header, payloads [][]byte) {
	defer func() { recover() }() // this will make sure we don't crash on out of bounds on byzantine messages
	status := gossipmessages.BenchmarkConsensusStatusReader(payloads[0])
	senderSignature := gossipmessages.SenderSignatureReader(payloads[1])
	for _, l := range s.benchmarkConsensusHandlers {
		l.HandleBenchmarkConsensusCommitted(&gossiptopics.BenchmarkConsensusCommittedInput{
			Message: &gossipmessages.BenchmarkConsensusCommittedMessage{
				Status: status,
				Sender: senderSignature,
			},
		})
	}
}
