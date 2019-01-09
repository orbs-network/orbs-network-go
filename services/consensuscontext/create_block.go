package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) createTransactionsBlock(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*protocol.TransactionsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createTxBlockTime.RecordSince(start)

	newBlockTimestamp := calculateNewBlockTimestamp(input.PrevBlockTimestamp, primitives.TimestampNano(time.Now().UnixNano()))

	proposedTransactions, err := s.fetchTransactions(ctx, input.CurrentBlockHeight, newBlockTimestamp, s.config.ConsensusContextMaximumTransactionsInBlock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch transactions for new block")
	}
	txCount := len(proposedTransactions.SignedTransactions)

	merkleTransactionsRoot, err := calculateTransactionsMerkleRoot(proposedTransactions.SignedTransactions)
	if err != nil {
		return nil, err
	}

	metaData := (&protocol.TransactionsBlockMetadataBuilder{}).Build()

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:            s.config.ProtocolVersion(),
			VirtualChainId:             s.config.VirtualChainId(),
			BlockHeight:                input.CurrentBlockHeight,
			PrevBlockHashPtr:           input.PrevBlockHash,
			Timestamp:                  newBlockTimestamp,
			TransactionsMerkleRootHash: merkleTransactionsRoot,
			MetadataHash:               digest.CalcTransactionMetaDataHash(metaData),
			NumSignedTransactions:      uint32(txCount),
		}).Build(),
		Metadata:           metaData,
		SignedTransactions: proposedTransactions.SignedTransactions,
		BlockProof:         (&protocol.TransactionsBlockProofBuilder{}).Build(),
	}

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))
	return txBlock, nil
}

func (s *service) createResultsBlock(ctx context.Context, input *services.RequestNewResultsBlockInput) (*protocol.ResultsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createResultsBlockTime.RecordSince(start)

	output, err := s.virtualMachine.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		SignedTransactions:    input.TransactionsBlock.SignedTransactions,
		CurrentBlockHeight:    input.CurrentBlockHeight,
		CurrentBlockTimestamp: input.TransactionsBlock.Header.Timestamp(),
	})
	if err != nil {
		return nil, err
	}

	merkleReceiptsRoot, err := calculateReceiptsMerkleRoot(output.TransactionReceipts)
	if err != nil {
		return nil, err
	}

	if input.CurrentBlockHeight == 0 {
		panic("CurrentBlockHeight, the block being closed, cannot be at height zero")
	}

	preExecutionStateRootHash := &services.GetStateHashOutput{}

	preExecutionStateRootHash, err = s.stateStorage.GetStateHash(ctx, &services.GetStateHashInput{
		BlockHeight: input.CurrentBlockHeight - 1,
	})
	if err != nil {
		return nil, err
	}

	stateDiffHash, err := calculateStateDiffMerkleRoot(output.ContractStateDiffs)
	if err != nil {
		return nil, err
	}

	rxBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			ProtocolVersion:                 primitives.ProtocolVersion(s.config.ProtocolVersion()),
			VirtualChainId:                  s.config.VirtualChainId(),
			BlockHeight:                     input.CurrentBlockHeight,
			PrevBlockHashPtr:                input.PrevBlockHash,
			Timestamp:                       input.TransactionsBlock.Header.Timestamp(),
			ReceiptsMerkleRootHash:          merkleReceiptsRoot,
			StateDiffHash:                   stateDiffHash,
			TransactionsBlockHashPtr:        digest.CalcTransactionsBlockHash(input.TransactionsBlock),
			PreExecutionStateMerkleRootHash: preExecutionStateRootHash.StateMerkleRootHash,
			NumTransactionReceipts:          uint32(len(output.TransactionReceipts)),
			NumContractStateDiffs:           uint32(len(output.ContractStateDiffs)),
		}).Build(),
		TransactionReceipts: output.TransactionReceipts,
		ContractStateDiffs:  output.ContractStateDiffs,
		BlockProof:          (&protocol.ResultsBlockProofBuilder{}).Build(),
	}
	return rxBlock, nil
}
