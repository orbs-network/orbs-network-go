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

	proposedTransactions, err := s.fetchTransactions(ctx, input.CurrentBlockHeight, newBlockTimestamp, s.config.ConsensusContextMaximumTransactionsInBlock(), s.config.ConsensusContextMinimumTransactionsInBlock(), s.config.ConsensusContextMinimalBlockTime())
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
