// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func txInputs(cfg config.ConsensusContextConfig) *services.ValidateTransactionsBlockInput {

	currentBlockHeight := primitives.BlockHeight(1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.ClientAddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := digest.CalcTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
	validMetadataHash := digest.CalcTransactionMetaDataHash(txMetadata.Build())
	validPrevBlock := builders.BlockPair().WithHeight(currentBlockHeight - 1).Build()
	validPrevBlockHash := digest.CalcTransactionsBlockHash(validPrevBlock.TransactionsBlock)
	validPrevBlockTimestamp := primitives.TimestampNano(time.Now().UnixNano() - 1000)

	// include only one transaction in block
	block := builders.
		BlockPair().
		WithHeight(currentBlockHeight).
		WithProtocolVersion(cfg.ProtocolVersion()).
		WithVirtualChainId(cfg.VirtualChainId()).
		WithTransactions(0).
		WithTransaction(transaction).
		WithPrevBlock(validPrevBlock).
		WithPrevBlockHash(validPrevBlockHash).
		WithMetadata(txMetadata).
		WithMetadataHash(validMetadataHash).
		WithTransactionsRootHash(txRootHashForValidBlock).
		Build()

	input := &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: currentBlockHeight,
		TransactionsBlock:  block.TransactionsBlock,
		PrevBlockHash:      validPrevBlockHash,
		PrevBlockTimestamp: validPrevBlockTimestamp,
	}

	return input
}

func TestValidateTransactionsBlockOnValidBlock(t *testing.T) {
	log := log.DefaultTestingLogger(t)
	metricFactory := metric.NewRegistry()
	cfg := config.ForConsensusContextTests(nil)
	txPool := &services.MockTransactionPool{}
	txPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil)

	s := consensuscontext.NewConsensusContext(
		txPool,
		&services.MockVirtualMachine{},
		&services.MockStateStorage{},
		cfg,
		log,
		metricFactory)

	input := txInputs(cfg)
	_, err := s.ValidateTransactionsBlock(context.Background(), input)
	require.NoError(t, err, "validation should succeed on valid block")
}

// TODO Merge with txInput
func rxInputs(cfg config.ConsensusContextConfig) *services.ValidateResultsBlockInput {

	currentBlockHeight := primitives.BlockHeight(1000)
	transaction := builders.TransferTransaction().WithAmountAndTargetAddress(10, builders.ClientAddressForEd25519SignerForTests(6)).Build()
	txMetadata := &protocol.TransactionsBlockMetadataBuilder{}
	txRootHashForValidBlock, _ := digest.CalcTransactionsMerkleRoot([]*protocol.SignedTransaction{transaction})
	validMetadataHash := digest.CalcTransactionMetaDataHash(txMetadata.Build())
	validPrevBlock := builders.BlockPair().WithHeight(currentBlockHeight - 1).Build()
	validPrevBlockHash := digest.CalcTransactionsBlockHash(validPrevBlock.TransactionsBlock)
	validPrevBlockTimestamp := primitives.TimestampNano(time.Now().UnixNano() - 1000)

	mockedPreExecutionStateMerkleRoot := hash.CalcSha256([]byte{1, 2, 3, 4, 5})

	block := builders.
		BlockPair().
		WithHeight(currentBlockHeight).
		WithProtocolVersion(cfg.ProtocolVersion()).
		WithVirtualChainId(cfg.VirtualChainId()).
		WithTransactions(3).
		WithPrevBlock(validPrevBlock).
		WithPrevBlockHash(validPrevBlockHash).
		WithMetadata(txMetadata).
		WithMetadataHash(validMetadataHash).
		WithTransactionsRootHash(txRootHashForValidBlock).
		WithStateDiffs(3).
		WithReceiptsForTransactions().
		Build()

	block.ResultsBlock.Header.MutateTransactionsBlockHashPtr(digest.CalcTransactionsBlockHash(block.TransactionsBlock))
	txReceiptsMerkleRoot, _ := digest.CalcReceiptsMerkleRoot(block.ResultsBlock.TransactionReceipts)
	block.ResultsBlock.Header.MutateReceiptsMerkleRootHash(txReceiptsMerkleRoot)
	stateDiffHash, _ := digest.CalcStateDiffHash(block.ResultsBlock.ContractStateDiffs)
	block.ResultsBlock.Header.MutateStateDiffHash(stateDiffHash)
	block.ResultsBlock.Header.MutatePreExecutionStateMerkleRootHash(mockedPreExecutionStateMerkleRoot)

	input := &services.ValidateResultsBlockInput{
		CurrentBlockHeight: currentBlockHeight,
		TransactionsBlock:  block.TransactionsBlock,
		ResultsBlock:       block.ResultsBlock,
		PrevBlockHash:      validPrevBlockHash,
		PrevBlockTimestamp: validPrevBlockTimestamp,
	}

	return input
}

func TestValidateResultsBlockOnValidBlock(t *testing.T) {
	log := log.DefaultTestingLogger(t)
	metricFactory := metric.NewRegistry()
	cfg := config.ForConsensusContextTests(nil)
	txPool := &services.MockTransactionPool{}
	txPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil)

	input := rxInputs(cfg)

	vm := &services.MockVirtualMachine{}
	output := &services.ProcessTransactionSetOutput{
		TransactionReceipts: input.ResultsBlock.TransactionReceipts,
		ContractStateDiffs:  input.ResultsBlock.ContractStateDiffs,
	}

	vm.When("ProcessTransactionSet", mock.Any, mock.Any).Return(output, nil)

	stateStorage := &services.MockStateStorage{}
	stateHashOutput := &services.GetStateHashOutput{
		StateMerkleRootHash: input.ResultsBlock.Header.PreExecutionStateMerkleRootHash(),
	}
	stateStorage.When("GetStateHash", mock.Any, mock.Any).Return(stateHashOutput, nil)

	s := consensuscontext.NewConsensusContext(
		txPool,
		vm,
		stateStorage,
		cfg,
		log,
		metricFactory)

	_, err := s.ValidateResultsBlock(context.Background(), input)
	require.NoError(t, err, "validation should succeed on valid block")
}
