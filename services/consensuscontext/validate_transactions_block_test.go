// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	testValidators "github.com/orbs-network/orbs-network-go/test/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func toTxValidatorContext(cfg config.ConsensusContextConfig) *txValidatorContext {

	block := testValidators.AStructurallyValidBlock()
	prevBlockHashCopy := make([]byte, 32)
	copy(prevBlockHashCopy, block.TransactionsBlock.Header.PrevBlockHashPtr())

	input := &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: block.TransactionsBlock.Header.BlockHeight(),
		TransactionsBlock:  block.TransactionsBlock, // fill in each test
		PrevBlockHash:      prevBlockHashCopy,
	}

	return &txValidatorContext{
		protocolVersion:        cfg.ProtocolVersion(),
		virtualChainId:         cfg.VirtualChainId(),
		allowedTimestampJitter: cfg.ConsensusContextSystemTimestampAllowedJitter(),
		input:                  input,
	}
}

func TestTransactionsBlockValidators(t *testing.T) {
	cfg := config.ForConsensusContextTests(nil)
	hash2 := hash.CalcSha256([]byte{2})
	falsyValidateTransactionOrdering := func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (
		*services.ValidateTransactionsForOrderingOutput, error) {
		return &services.ValidateTransactionsForOrderingOutput{}, errors.New("Some error")
	}

	t.Run("should return error for transaction block with incorrect protocol version", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateProtocolVersion(999); err != nil {
			t.Error(err)
		}
		err := validateTxProtocolVersion(context.Background(), vctx)
		require.Equal(t, ErrMismatchedProtocolVersion, errors.Cause(err), "validation should fail on incorrect protocol version", err)
	})

	t.Run("should return error for transaction block with incorrect virtual chain", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateVirtualChainId(999); err != nil {
			t.Error(err)
		}
		err := validateTxVirtualChainID(context.Background(), vctx)
		require.Equal(t, ErrMismatchedVirtualChainID, errors.Cause(err), "validation should fail on incorrect virtual chain", err)
	})

	t.Run("should return error for transaction block with incorrect block height", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutateBlockHeight(1); err != nil {
			t.Error(err)
		}

		err := validateTxBlockHeight(context.Background(), vctx)
		require.Equal(t, ErrMismatchedBlockHeight, errors.Cause(err), "validation should fail on incorrect block height", err)
	})

	t.Run("should return error for transaction block with incorrect prev block hash", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		if err := vctx.input.TransactionsBlock.Header.MutatePrevBlockHashPtr(hash2); err != nil {
			t.Error(err)
		}
		err := validateTxPrevBlockHashPtr(context.Background(), vctx)
		require.Equal(t, ErrMismatchedPrevBlockHash, errors.Cause(err), "validation should fail on incorrect prev block hash", err)
	})

	t.Run("should return error for transaction block with failing tx ordering validation", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		vctx.txOrderValidator = falsyValidateTransactionOrdering
		err := validateTxTransactionOrdering(context.Background(), vctx)
		require.Equal(t, ErrFailedTransactionOrdering, errors.Cause(err), "validation should fail on failing tx ordering validation", err)
	})
}

func TestIsValidBlockTimestamp(t *testing.T) {

	jitter := 2 * time.Second
	tests := []struct {
		name                        string
		currentBlockTimestampOffset time.Duration
		prevBlockTimestampOffset    time.Duration
		expectedToPass              bool
	}{
		{
			"Current block has valid timestamp",
			1 * time.Second,
			-3 * time.Second,
			true,
		},
		{
			"Current block is too far in the past",
			-3 * time.Second,
			-6 * time.Second,
			false,
		},
		{
			"Current block is too far in the future",
			3 * time.Second,
			-6 * time.Second,
			false,
		},
		{
			"Current block is older than prev block",
			-2 * time.Second,
			-1 * time.Second,
			false,
		},
		{
			"Current block is as old as prev block",
			-2 * time.Second,
			-2 * time.Second,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			currentBlockTimestamp := primitives.TimestampNano(now.Add(tt.currentBlockTimestampOffset).UnixNano())
			prevBlockTimestamp := primitives.TimestampNano(now.Add(tt.prevBlockTimestampOffset).UnixNano())
			err := isValidBlockTimestamp(currentBlockTimestamp, prevBlockTimestamp, now, jitter)
			if tt.expectedToPass {
				require.NoError(t, err, tt.name)
			} else {
				require.Error(t, err, tt.name)
			}
		})
	}
}
