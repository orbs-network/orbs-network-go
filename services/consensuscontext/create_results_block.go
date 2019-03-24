// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

func (s *service) createResultsBlock(ctx context.Context, input *services.RequestNewResultsBlockInput) (*protocol.ResultsBlockContainer, error) {
	start := time.Now()
	defer s.metrics.createResultsBlockTime.RecordSince(start)

	output, err := s.virtualMachine.ProcessTransactionSet(ctx, &services.ProcessTransactionSetInput{
		SignedTransactions:    input.TransactionsBlock.SignedTransactions,
		CurrentBlockHeight:    input.CurrentBlockHeight,
		CurrentBlockTimestamp: input.TransactionsBlock.Header.Timestamp(),
	})
	s.metrics.processTransactionsSeInCreateResultsBlock.RecordSince(start)
	if err != nil {
		return nil, err
	}

	merkleReceiptsRoot, err := digest.CalcReceiptsMerkleRoot(output.TransactionReceipts)
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

	stateDiffHash, err := digest.CalcStateDiffHash(output.ContractStateDiffs)
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
