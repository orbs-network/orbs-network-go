// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Triggers"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) createTransactionsBlock(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*protocol.TransactionsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createTxBlockTime.RecordSince(start)

	proposedTransactions, err := s.fetchTransactions(ctx, input.CurrentBlockHeight, input.PrevBlockTimestamp, s.config.ConsensusContextMaximumTransactionsInBlock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch transactions for new block")
	}

	newBlockTimestamp := proposedTransactions.ProposedBlockTimestamp
	if newBlockTimestamp == 0 {
		return nil, errors.New("transactions pool GetTransactionsForOrdering returned proposed block timestamp of zero")
	}

	transactionsForBlock := s.updateTransactions(proposedTransactions.SignedTransactions, proposedTransactions.ProposedBlockTimestamp)
	txCount := len(transactionsForBlock)

	merkleTransactionsRoot, err := digest.CalcTransactionsMerkleRoot(transactionsForBlock)
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
		SignedTransactions: transactionsForBlock,
		BlockProof:         (&protocol.TransactionsBlockProofBuilder{}).Build(),
	}

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))
	return txBlock, nil
}

func (s *service) createTriggerTransaction(blockTime primitives.TimestampNano) *protocol.SignedTransaction {
	return (&protocol.SignedTransactionBuilder{
		Transaction: &protocol.TransactionBuilder{
			ProtocolVersion: s.config.ProtocolVersion(),
			VirtualChainId:  s.config.VirtualChainId(),
			Timestamp:       blockTime,
			ContractName:    primitives.ContractName(triggers_systemcontract.CONTRACT_NAME),
			MethodName:      primitives.MethodName(triggers_systemcontract.METHOD_TRIGGER),
		},
	}).Build()
}

func (s *service) updateTransactions(txs []*protocol.SignedTransaction, blockTime primitives.TimestampNano) []*protocol.SignedTransaction {
	if s.config.ConsensusContextTriggersEnabled() {
		txs = append(txs, s.createTriggerTransaction(blockTime))
	}
	return txs
}
