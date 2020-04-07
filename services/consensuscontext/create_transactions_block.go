// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/Triggers"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

func (s *service) createTransactionsBlock(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*protocol.TransactionsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createTxBlockTime.RecordSince(start)

	proposedReferenceTime, err  := s.proposeBlockReferenceTime(ctx, input.PrevBlockReferenceTime)
	if err != nil {
		return nil, err
	}

	proposedProtocolVersion := s.management.GetProtocolVersion(ctx, proposedReferenceTime)

	proposedTransactions, err := s.fetchTransactions(ctx, proposedProtocolVersion, input.CurrentBlockHeight, input.PrevBlockTimestamp, proposedReferenceTime, s.config.ConsensusContextMaximumTransactionsInBlock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch transactions for new block")
	}

	proposedBlockTimestamp := proposedTransactions.ProposedBlockTimestamp
	if proposedBlockTimestamp == 0 {
		return nil, errors.New("transactions pool GetTransactionsForOrdering returned proposed block timestamp of zero")
	}

	transactionsForBlock := s.updateTransactions(proposedTransactions.SignedTransactions, proposedProtocolVersion, proposedBlockTimestamp)
	txCount := len(transactionsForBlock)

	merkleTransactionsRoot, err := digest.CalcTransactionsMerkleRoot(transactionsForBlock)
	if err != nil {
		return nil, err
	}

	metaData := (&protocol.TransactionsBlockMetadataBuilder{}).Build()

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:            proposedProtocolVersion,
			VirtualChainId:             s.config.VirtualChainId(),
			BlockHeight:                input.CurrentBlockHeight,
			PrevBlockHashPtr:           input.PrevBlockHash,
			Timestamp:                  proposedBlockTimestamp,
			TransactionsMerkleRootHash: merkleTransactionsRoot,
			MetadataHash:               digest.CalcTransactionMetaDataHash(metaData),
			NumSignedTransactions:      uint32(txCount),
			BlockProposerAddress:       input.BlockProposerAddress,
			ReferenceTime:              proposedReferenceTime,
		}).Build(),
		Metadata:           metaData,
		SignedTransactions: transactionsForBlock,
		BlockProof:         (&protocol.TransactionsBlockProofBuilder{}).Build(),
	}

	s.metrics.transactionsRate.Measure(int64(len(txBlock.SignedTransactions)))
	return txBlock, nil
}

func (s *service) proposeBlockReferenceTime(ctx context.Context, prevReferenceTime primitives.TimestampSeconds) (primitives.TimestampSeconds, error) {
	proposedReferenceTime := s.management.GetCurrentReference(ctx)
	if err := validateProposeBlockReferenceTime(prevReferenceTime, proposedReferenceTime,
		s.management.GetCurrentReference(ctx), s.config.ManagementConsensusGraceTimeout()); err != nil {
		return 0, err
	}

	// NOTE: network live and subscription is done in vm.pre-order to allow empty blocks to close.
	return proposedReferenceTime, nil
}

func (s *service) fetchTransactions(ctx context.Context, blockProtocolVersion primitives.ProtocolVersion, currentBlockHeight primitives.BlockHeight, prevBlockTimestamp primitives.TimestampNano, currentBlockReferenceTime primitives.TimestampSeconds, maxNumberOfTransactions uint32) (*services.GetTransactionsForOrderingOutput, error) {
	input := &services.GetTransactionsForOrderingInput{
		BlockProtocolVersion:      blockProtocolVersion,
		CurrentBlockHeight:        currentBlockHeight,
		PrevBlockTimestamp:        prevBlockTimestamp,
		CurrentBlockReferenceTime: currentBlockReferenceTime,
		MaxTransactionsSetSizeKb:  0, // TODO(v1): either fill in or delete from spec
		MaxNumberOfTransactions:   maxNumberOfTransactions,
	}

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(ctx, input)
	if err != nil {
		return nil, err
	}

	return proposedTransactions, nil
}

func (s *service) createTriggerTransaction(protocolVersion primitives.ProtocolVersion, blockTime primitives.TimestampNano) *protocol.SignedTransaction {
	return (&protocol.SignedTransactionBuilder{
		Transaction: &protocol.TransactionBuilder{
			ProtocolVersion: protocolVersion,
			VirtualChainId:  s.config.VirtualChainId(),
			Timestamp:       blockTime,
			ContractName:    primitives.ContractName(triggers_systemcontract.CONTRACT_NAME),
			MethodName:      primitives.MethodName(triggers_systemcontract.METHOD_TRIGGER),
		},
	}).Build()
}

func (s *service) updateTransactions(txs []*protocol.SignedTransaction, protocolVersion primitives.ProtocolVersion, blockTime primitives.TimestampNano) []*protocol.SignedTransaction {
	if s.config.ConsensusContextTriggersEnabled() {
		txs = append(txs, s.createTriggerTransaction(protocolVersion, blockTime))
	}
	return txs
}
