package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) createTransactionsBlock(blockHeight primitives.BlockHeight, prevBlockHash primitives.Sha256) (*protocol.TransactionsBlockContainer, error) {
	meter := s.logger.Meter("tx-block-creation")
	defer meter.Done()
	start := time.Now()
	defer s.metrics.createTxBlock.RecordNanosSince(start)


	proposedTransactions, err := s.fetchTransactions(s.config.ConsensusContextMaximumTransactionsInBlock(), s.config.ConsensusContextMinimumTransactionsInBlock(), s.config.ConsensusContextMinimalBlockDelay())
	if err != nil {
		return nil, err
	}
	txCount := len(proposedTransactions.SignedTransactions)

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:       blockstorage.ProtocolVersion,
			BlockHeight:           blockHeight,
			//Timestamp: 			   primitives.TimestampNano(time.Now().UnixNano()),
			PrevBlockHashPtr:      prevBlockHash,
			NumSignedTransactions: uint32(txCount),
		}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: proposedTransactions.SignedTransactions,
		BlockProof:         nil,
	}
	return txBlock, nil
}

func (s *service) createResultsBlock(blockHeight primitives.BlockHeight, prevBlockHash primitives.Sha256, transactionsBlock *protocol.TransactionsBlockContainer) (*protocol.ResultsBlockContainer, error) {
	meter := s.logger.Meter("rs-block-creation")
	defer meter.Done()
	defer s.metrics.createResultsBlock.RecordNanosSince(time.Now())

	output, err := s.virtualMachine.ProcessTransactionSet(&services.ProcessTransactionSetInput{
		BlockHeight:        blockHeight,
		SignedTransactions: transactionsBlock.SignedTransactions,
	})
	if err != nil {
		return nil, err
	}

	rxBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			ProtocolVersion:          blockstorage.ProtocolVersion,
			BlockHeight:              blockHeight,
			PrevBlockHashPtr:         prevBlockHash,
			TransactionsBlockHashPtr: digest.CalcTransactionsBlockHash(transactionsBlock),
			NumTransactionReceipts:   uint32(len(output.TransactionReceipts)),
			NumContractStateDiffs:    uint32(len(output.ContractStateDiffs)),
		}).Build(),
		TransactionReceipts: output.TransactionReceipts,
		ContractStateDiffs:  output.ContractStateDiffs,
		BlockProof:          nil,
	}
	return rxBlock, nil
}
