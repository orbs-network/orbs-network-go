// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func aMockValidateTransactionsBlockThatReturnsSuccess(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
	return nil, nil
}

func aMockValidateTransactionsBlockThatReturnsError(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {
	return nil, errors.New("Some error")
}

func aMockValidateResultsBlockThatReturnsSuccess(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	return nil, nil
}

func aMockValidateResultsBlockThatReturnsError(ctx context.Context, input *services.ValidateResultsBlockInput) (*services.ValidateResultsBlockOutput, error) {
	return nil, errors.New("Some error")
}

func aMockValidateBlockHashThatReturnsSuccess(blockHash primitives.Sha256, tx *protocol.TransactionsBlockContainer, rx *protocol.ResultsBlockContainer) error {
	return nil
}

func aMockValidateBlockHashThatReturnsError(blockHash primitives.Sha256, tx *protocol.TransactionsBlockContainer, rx *protocol.ResultsBlockContainer) error {
	return errors.New("Some error")
}

// We don't care about the correctness or incorrectness of inputs because we mock the functions ValidateTransactionsBlock()
// and ValidateResultsBlock() that actually test those inputs.
// We only test the glue that holds them together. These are tests for these 2 functions in the same package where they are defined.

func TestValidateBlockProposal_HappyPath(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			block := builders.BlockPairBuilder().Build()
			prevBlock := builders.BlockPairBuilder().Build()
			require.NoError(t, validateBlockProposalInternal(ctx, ToLeanHelixBlock(block), []byte{1, 2, 3, 4}, hash.Make32EmptyBytes(), ToLeanHelixBlock(prevBlock), &validateBlockProposalContext{
				validateTransactionsBlock: aMockValidateTransactionsBlockThatReturnsSuccess,
				validateResultsBlock:      aMockValidateResultsBlockThatReturnsSuccess,
				validateBlockHash:         aMockValidateBlockHashThatReturnsSuccess,
				logger:                    harness.Logger,
			}), "should return true when ValidateTransactionsBlock() and ValidateResultsBlock() are successful")
		})
	})
}

func TestValidateBlockProposal_FailsOnErrorInTransactionsBlock(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			block := builders.BlockPairBuilder().Build()
			prevBlock := builders.BlockPairBuilder().Build()
			require.Error(t, validateBlockProposalInternal(ctx, ToLeanHelixBlock(block), []byte{1, 2, 3, 4}, hash.Make32EmptyBytes(), ToLeanHelixBlock(prevBlock), &validateBlockProposalContext{
				validateTransactionsBlock: aMockValidateTransactionsBlockThatReturnsError,
				validateResultsBlock:      aMockValidateResultsBlockThatReturnsSuccess,
				validateBlockHash:         aMockValidateBlockHashThatReturnsSuccess,
				logger:                    harness.Logger,
			}), "should return false when ValidateTransactionsBlock() returns an error")
		})
	})
}

func TestValidateBlockProposal_FailsOnErrorInResultsBlock(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			block := builders.BlockPairBuilder().Build()
			prevBlock := builders.BlockPairBuilder().Build()
			require.Error(t, validateBlockProposalInternal(ctx, ToLeanHelixBlock(block), []byte{1, 2, 3, 4}, hash.Make32EmptyBytes(), ToLeanHelixBlock(prevBlock), &validateBlockProposalContext{
				validateTransactionsBlock: aMockValidateTransactionsBlockThatReturnsSuccess,
				validateResultsBlock:      aMockValidateResultsBlockThatReturnsError,
				validateBlockHash:         aMockValidateBlockHashThatReturnsSuccess,
				logger:                    harness.Logger,
			}), "should return false when ValidateResultsBlock() returns an error")
		})
	})
}

func TestValidateBlockProposal_FailsOnErrorInValidateBlockHash(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(harness *with.LoggingHarness) {
			block := builders.BlockPairBuilder().Build()
			prevBlock := builders.BlockPairBuilder().Build()
			require.Error(t, validateBlockProposalInternal(ctx, ToLeanHelixBlock(block), []byte{1, 2, 3, 4}, hash.Make32EmptyBytes(), ToLeanHelixBlock(prevBlock), &validateBlockProposalContext{
				validateTransactionsBlock: aMockValidateTransactionsBlockThatReturnsSuccess,
				validateResultsBlock:      aMockValidateResultsBlockThatReturnsSuccess,
				validateBlockHash:         aMockValidateBlockHashThatReturnsError,
				logger:                    harness.Logger,
			}), "should return false when ValidateBlockHash() returns an error")
		})
	})
}
