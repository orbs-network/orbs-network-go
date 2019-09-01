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
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func txInputs(cfg config.ConsensusContextConfig) *services.ValidateTransactionsBlockInput {
	block := builders.BlockPairBuilder().WithCfg(cfg).Build()

	input := &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: block.TransactionsBlock.Header.BlockHeight(),
		TransactionsBlock:  block.TransactionsBlock,
		PrevBlockHash:      block.TransactionsBlock.Header.PrevBlockHashPtr(),
		PrevBlockTimestamp: block.TransactionsBlock.Header.Timestamp() - 1000,
	}

	return input
}

func TestValidateTransactionsBlockOnValidBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			s := newHarness(harness.Logger, true)
			s.transactionPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil)
			input := txInputs(s.config)

			_, err := s.service.ValidateTransactionsBlock(ctx, input)
			require.NoError(t, err, "validation should succeed on valid block")
		})
	})
}

func TestValidateTransactionsBlockOnValidBlockWithoutTrigger(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			s := newHarness(harness.Logger, false)
			s.transactionPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil)
			input := txInputs(s.config)

			_, err := s.service.ValidateTransactionsBlock(ctx, input)
			require.NoError(t, err, "validation should fail when missing trigger")
		})
	})
}

func rxInputs(cfg config.ConsensusContextConfig) *services.ValidateResultsBlockInput {
	block := builders.BlockPairBuilder().WithCfg(cfg).Build()

	input := &services.ValidateResultsBlockInput{
		CurrentBlockHeight: block.TransactionsBlock.Header.BlockHeight(),
		TransactionsBlock:  block.TransactionsBlock,
		PrevBlockHash:      block.TransactionsBlock.Header.PrevBlockHashPtr(),
		PrevBlockTimestamp: block.TransactionsBlock.Header.Timestamp() - 1000,
		ResultsBlock:       block.ResultsBlock,
	}

	return input
}

func TestValidateResultsBlockOnValidBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			s := newHarness(harness.Logger, false)

			input := rxInputs(s.config)
			s.transactionPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, nil)

			output := &services.ProcessTransactionSetOutput{
				TransactionReceipts: input.ResultsBlock.TransactionReceipts,
				ContractStateDiffs:  input.ResultsBlock.ContractStateDiffs,
			}
			s.virtualMachine.When("ProcessTransactionSet", mock.Any, mock.Any).Return(output, nil)

			stateHashOutput := &services.GetStateHashOutput{
				StateMerkleRootHash: input.ResultsBlock.Header.PreExecutionStateMerkleRootHash(),
			}
			s.stateStorage.When("GetStateHash", mock.Any, mock.Any).Return(stateHashOutput, nil)

			_, err := s.service.ValidateResultsBlock(ctx, input)
			require.NoError(t, err, "validation should succeed on valid block")
		})
	})
}
