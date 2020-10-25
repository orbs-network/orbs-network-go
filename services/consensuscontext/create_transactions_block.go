// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
    "github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/Triggers"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"time"
)

func (s *service) createTransactionsBlock(ctx context.Context, input *services.RequestNewTransactionsBlockInput) (*protocol.TransactionsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createTxBlockTime.RecordSince(start)

	prevBlockReferenceTime, err := s.adjustPrevReference(ctx, input.CurrentBlockHeight, input.PrevBlockReferenceTime) // For completeness, can't really fail
	if err != nil {
		return nil, errors.Wrapf(ErrFailedGenesisRefTime, "CreateTransactionsBlock failed genesis time %s", err)
	}

	proposedReferenceTime, err := s.proposeBlockReferenceTime(ctx, prevBlockReferenceTime)
	if err != nil {
		return nil, err
	}

	proposedProtocolVersion, err := s.management.GetProtocolVersion(ctx, &services.GetProtocolVersionInput{Reference: proposedReferenceTime})
	if err != nil {
		s.logger.Error("management.GetProtocolVersion should not return error", log.Error(err))
		return nil, err
	}

	proposedTransactions, err := s.fetchTransactions(ctx, proposedProtocolVersion.ProtocolVersion, input.CurrentBlockHeight, input.PrevBlockTimestamp, proposedReferenceTime, s.config.ConsensusContextMaximumTransactionsInBlock())
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch transactions for new block")
	}

	proposedBlockTimestamp := proposedTransactions.ProposedBlockTimestamp
	if proposedBlockTimestamp == 0 {
		return nil, errors.New("transactions pool GetTransactionsForOrdering returned proposed block timestamp of zero")
	}

	transactionsForBlock := s.updateTransactions(proposedTransactions.SignedTransactions, proposedBlockTimestamp)
	txCount := len(transactionsForBlock)

	merkleTransactionsRoot, err := digest.CalcTransactionsMerkleRoot(transactionsForBlock)
	if err != nil {
		return nil, err
	}

	metaData := (&protocol.TransactionsBlockMetadataBuilder{}).Build()

	txBlock := &protocol.TransactionsBlockContainer{
		Header: (&protocol.TransactionsBlockHeaderBuilder{
			ProtocolVersion:            proposedProtocolVersion.ProtocolVersion,
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
	leaderReferenceTime, err := s.management.GetCurrentReference(ctx, &services.GetCurrentReferenceInput{SystemTime: prevReferenceTime})
	if err != nil {
		s.logger.Error("management.GetCurrentReference should not return error", log.Error(err))
	}

	if leaderReferenceTime.CurrentReference < prevReferenceTime {
		return 0, errors.Errorf("leader reference time %d (before grace adjustment) is not upto date compared to previous block reference time %d", leaderReferenceTime.CurrentReference, prevReferenceTime)
	}

	proposedReferenceTime := leaderReferenceTime.CurrentReference - primitives.TimestampSeconds(s.config.ManagementConsensusGraceTimeout()/time.Second)
	if prevReferenceTime > proposedReferenceTime {
		proposedReferenceTime = prevReferenceTime
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

func (s *service) createTriggerTransaction(blockTime primitives.TimestampNano) *protocol.SignedTransaction {
	return (&protocol.SignedTransactionBuilder{
		Transaction: &protocol.TransactionBuilder{
			ProtocolVersion: config.MAXIMAL_CLIENT_PROTOCOL_VERSION,
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
