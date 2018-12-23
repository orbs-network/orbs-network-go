package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) createTransactionsBlock(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*protocol.TransactionsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createTxBlockTime.RecordSince(start)

	proposedTransactions, err := s.fetchTransactions(ctx, input.BlockHeight, s.config.ConsensusContextMaximumTransactionsInBlock(), s.config.ConsensusContextMinimumTransactionsInBlock(), s.config.ConsensusContextMinimalBlockTime())
	if err != nil {
		return nil, err
	}
	txCount := len(proposedTransactions.SignedTransactions)

	merkleTransactionsRoot, err := calculateTransactionsRootHash(proposedTransactions.SignedTransactions)
	if err != nil {
		return nil, err
	}

	timestamp := calculateNewBlockTimestamp(input.PrevBlockTimestamp, primitives.TimestampNano(time.Now().UnixNano()))

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:       primitives.ProtocolVersion(s.config.ProtocolVersion()),
			VirtualChainId:        s.config.VirtualChainId(),
			BlockHeight:           input.BlockHeight,
			PrevBlockHashPtr:      input.PrevBlockHash,
			Timestamp:             timestamp,
			TransactionsRootHash:  primitives.MerkleSha256(merkleTransactionsRoot),
			MetadataHash:          nil,
			NumSignedTransactions: uint32(txCount),
		}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: proposedTransactions.SignedTransactions,
		BlockProof:         (&protocol.TransactionsBlockProofBuilder{}).Build(),
	}

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))
	return txBlock, nil
}

func calculateTransactionsRootHash(txs []*protocol.SignedTransaction) (primitives.Sha256, error) {
	hashes := digest.CalcSignedTxHashes(txs)
	return merkle.CalculateOrderedTreeRoot(hashes), nil
}

func calculateReceiptsRootHash(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	hashes := digest.CalcReceiptHashes(receipts)
	return merkle.CalculateOrderedTreeRoot(hashes), nil
}

func calculatePrevBlockHashPtr(txBlock *protocol.TransactionsBlockContainer) primitives.Sha256 {
	return digest.CalcTransactionsBlockHash(txBlock)
}

func (s *service) createResultsBlock(ctx context.Context, input *services.RequestNewResultsBlockInput) (*protocol.ResultsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createResultsBlockTime.RecordSince(start)

	output, err := s.virtualMachine.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		BlockHeight:        input.BlockHeight,
		SignedTransactions: input.TransactionsBlock.SignedTransactions,
	})
	if err != nil {
		return nil, err
	}

	merkleReceiptsRoot, err := calculateReceiptsRootHash(output.TransactionReceipts)
	if err != nil {
		return nil, err
	}

	// TODO: handle genesis block at height 0
	preExecutionStateRootHash := &services.GetStateHashOutput{}
	if input.BlockHeight > 0 {
		preExecutionStateRootHash, err = s.stateStorage.GetStateHash(ctx, &services.GetStateHashInput{
			BlockHeight: input.BlockHeight - 1,
		})
		if err != nil {
			return nil, err
		}
	}

	stateDiffHash, err := calculateStateDiffHash(output.ContractStateDiffs)
	if err != nil {
		return nil, err
	}

	rxBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			ProtocolVersion:             primitives.ProtocolVersion(s.config.ProtocolVersion()),
			VirtualChainId:              s.config.VirtualChainId(),
			BlockHeight:                 input.BlockHeight,
			PrevBlockHashPtr:            input.PrevBlockHash,
			Timestamp:                   input.TransactionsBlock.Header.Timestamp(),
			ReceiptsRootHash:            primitives.MerkleSha256(merkleReceiptsRoot),
			StateDiffHash:               stateDiffHash,
			TransactionsBlockHashPtr:    digest.CalcTransactionsBlockHash(input.TransactionsBlock),
			PreExecutionStateRootHash:   preExecutionStateRootHash.StateRootHash,
			TransactionsBloomFilterHash: nil,
			NumTransactionReceipts:      uint32(len(output.TransactionReceipts)),
			NumContractStateDiffs:       uint32(len(output.ContractStateDiffs)),
		}).Build(),
		TransactionsBloomFilter: (&protocol.TransactionsBloomFilterBuilder{
			TxhashBloomFilter:    nil,
			TimestampBloomFilter: nil,
		}).Build(),
		TransactionReceipts: output.TransactionReceipts,
		ContractStateDiffs:  output.ContractStateDiffs,
		BlockProof:          (&protocol.ResultsBlockProofBuilder{}).Build(),
	}
	return rxBlock, nil
}
func calculateStateDiffHash(diffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {
	// TODO IMPL THIS https://tree.taiga.io/project/orbs-network/us/535
	return hash.CalcSha256([]byte{1, 2, 3, 4, 5, 6, 6, 7, 8}), nil
}

func calculateReceiptsRootHash(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error) {
	rptHashValues := make([]primitives.Sha256, len(receipts))
	for i := 0; i < len(receipts); i++ {
		rptHashValues[i] = receipts[i].Txhash()
	}
	return merkle.CalculateOrderedTreeRoot(rptHashValues), nil
}

func calculateNewBlockTimestamp(prevBlockTimestamp primitives.TimestampNano, now primitives.TimestampNano) primitives.TimestampNano {
	if now > prevBlockTimestamp {
		return now + 1
	}
	return prevBlockTimestamp + 1
}
