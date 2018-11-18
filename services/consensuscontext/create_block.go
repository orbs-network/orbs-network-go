package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/statestorage/merkle"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"strconv"
	"time"
)

func (s *service) createTransactionsBlock(ctx context.Context, blockHeight primitives.BlockHeight, prevBlockHash primitives.Sha256) (*protocol.TransactionsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createTxBlockTime.RecordSince(start)

	proposedTransactions, err := s.fetchTransactions(ctx, s.config.ConsensusContextMaximumTransactionsInBlock(), s.config.ConsensusContextMinimumTransactionsInBlock(), s.config.ConsensusContextMinimalBlockTime())
	if err != nil {
		return nil, err
	}
	txCount := len(proposedTransactions.SignedTransactions)

	merkleTransactionsRoot, err := CalculateTransactionsRootHash(proposedTransactions.SignedTransactions)
	if err != nil {
		return nil, err
	}

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:       primitives.ProtocolVersion(s.config.ProtocolVersion()),
			VirtualChainId:        s.config.VirtualChainId(),
			BlockHeight:           blockHeight,
			PrevBlockHashPtr:      prevBlockHash,
			Timestamp:             primitives.TimestampNano(time.Now().UnixNano()),
			TransactionsRootHash:  merkleTransactionsRoot,
			MetadataHash:          nil,
			NumSignedTransactions: uint32(txCount),
		}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: proposedTransactions.SignedTransactions,
		BlockProof:         nil,
	}

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))
	return txBlock, nil
}

func CalculateTransactionsRootHash(txs []*protocol.SignedTransaction) (primitives.MerkleSha256, error) {
	forest, root := merkle.NewForest()
	diffs := make([]*merkle.MerkleDiff, len(txs))
	for i := 0; i < len(txs); i++ {
		txHash := digest.CalcTxHash(txs[i].Transaction())
		diffs[i] = &merkle.MerkleDiff{
			Key:   []byte(strconv.Itoa(i)), // no need to be overly smart here
			Value: txHash,
		}
	}
	return forest.Update(root, diffs)
}

func CalculatePrevBlockHashPtr(txBlock *protocol.TransactionsBlockContainer) primitives.Sha256 {
	return digest.CalcTransactionsBlockHash(txBlock)
}

func (s *service) createResultsBlock(ctx context.Context, blockHeight primitives.BlockHeight, prevBlockHash primitives.Sha256, transactionsBlock *protocol.TransactionsBlockContainer) (*protocol.ResultsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createResultsBlockTime.RecordSince(start)

	output, err := s.virtualMachine.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		BlockHeight:        blockHeight,
		SignedTransactions: transactionsBlock.SignedTransactions,
	})
	if err != nil {
		return nil, err
	}
	merkleReceiptsRoot, err := calculateReceiptsRootHash(output.TransactionReceipts)
	if err != nil {
		return nil, err
	}

	// TODO Waiting for state-storage fix: internal sync does not yet update the state storage when committing blocks
	//preExecutionStateRootHash, err := s.stateStorage.GetStateHash(ctx, &services.GetStateHashInput{
	//	BlockHeight: blockHeight - 1,
	//})

	if err != nil {
		return nil, err
	}
	stateDiffHash, err := calculateStateDiffHash(output.ContractStateDiffs)
	if err != nil {
		return nil, err
	}

	rxBlock := &protocol.ResultsBlockContainer{
		Header: (&protocol.ResultsBlockHeaderBuilder{
			ProtocolVersion:           primitives.ProtocolVersion(s.config.ProtocolVersion()),
			VirtualChainId:            s.config.VirtualChainId(),
			BlockHeight:               blockHeight,
			PrevBlockHashPtr:          prevBlockHash,
			Timestamp:                 primitives.TimestampNano(time.Now().UnixNano()),
			ReceiptsRootHash:          merkleReceiptsRoot,
			StateDiffHash:             stateDiffHash,
			TransactionsBlockHashPtr:  digest.CalcTransactionsBlockHash(transactionsBlock),
			PreExecutionStateRootHash: nil,
			TxhashBloomFilter:         nil, // TODO ODEDW to decide
			TimestampBloomFilter:      nil, // TODO ODEDW to decide
			NumTransactionReceipts:    uint32(len(output.TransactionReceipts)),
			NumContractStateDiffs:     uint32(len(output.ContractStateDiffs)),
		}).Build(),
		TransactionReceipts: output.TransactionReceipts,
		ContractStateDiffs:  output.ContractStateDiffs,
		BlockProof:          nil,
	}
	return rxBlock, nil
}
func calculateStateDiffHash(diffs []*protocol.ContractStateDiff) (primitives.Sha256, error) {

	// TODO Just a placeholder for now, ODEDW to decide
	return hash.CalcSha256([]byte{1, 2, 3, 4, 5, 6, 6, 7, 8}), nil
}

func calculateReceiptsRootHash(receipts []*protocol.TransactionReceipt) (primitives.MerkleSha256, error) {
	forest, root := merkle.NewForest()
	diffs := make([]*merkle.MerkleDiff, len(receipts))
	for i := 0; i < len(receipts); i++ {
		diffs[i] = &merkle.MerkleDiff{
			Key:   []byte(strconv.Itoa(i)), // no need to be overly smart here
			Value: receipts[i].Txhash(),
		}
	}
	return forest.Update(root, diffs)

}
