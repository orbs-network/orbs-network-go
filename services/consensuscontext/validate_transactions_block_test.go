// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func toTxValidatorContext(cfg config.ConsensusContextConfig) *txValidatorContext {
	block := builders.BlockPairBuilder().Build()
	prevBlockHashCopy := make([]byte, 32)
	copy(prevBlockHashCopy, block.TransactionsBlock.Header.PrevBlockHashPtr())

	input := &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: block.TransactionsBlock.Header.BlockHeight(),
		TransactionsBlock:  block.TransactionsBlock,                                        // fill in each test
		PrevBlockTimestamp: primitives.TimestampNano(time.Now().Add(time.Hour).UnixNano()), // this is intentionally set in the future to fail timestamp tests
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
	cfg := config.ForConsensusContextTests(nil, false)
	hash2 := hash.CalcSha256([]byte{2})

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

	t.Run("should return error for invalid timestamp of block", func(t *testing.T) {
		vctx := toTxValidatorContext(cfg)
		err := validateTxTransactionsBlockTimestamp(context.Background(), vctx)
		require.Error(t, err, "validation should fail on invalid timestamp of block")
	})
}

func TestConsensusContextValidateTransactionsBlockTriggerTransactionNotForwardedToPreOrder(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		block := builders.BlockPairBuilder().Build()
		cfg := config.ForConsensusContextTests(nil, true)
		txPool := &services.MockTransactionPool{}
		txPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) {
			require.Equal(t, len(block.TransactionsBlock.SignedTransactions)-1, len(input.SignedTransactions))
		}).Return(nil, nil)
		err := validateTxTransactionOrdering(ctx, cfg, txPool.ValidateTransactionsForOrdering, block.TransactionsBlock)
		require.NoError(t, err)

		ok, err := txPool.Verify()
		require.True(t, ok)
		require.NoError(t, err)
	})
}

func TestConsensusContextValidateTransactionsBlockTriggerDisabledTransactionNotRemovedForForwardedToPreOrder(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		cfg := config.ForConsensusContextTests(nil, false)
		block := builders.BlockPairBuilder().WithCfg(cfg).Build()
		txPool := &services.MockTransactionPool{}
		txPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) {
			require.Equal(t, len(block.TransactionsBlock.SignedTransactions), len(input.SignedTransactions))
		}).Return(nil, nil)
		err := validateTxTransactionOrdering(ctx, cfg, txPool.ValidateTransactionsForOrdering, block.TransactionsBlock)
		require.NoError(t, err)

		ok, err := txPool.Verify()
		require.True(t, ok)
		require.NoError(t, err)
	})
}

func TestConsensusContextValidateTransactionsBlock_ForForwardedToPreOrderErrors(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		cfg := config.ForConsensusContextTests(nil, false)
		block := builders.BlockPairBuilder().WithCfg(cfg).Build()
		txPool := &services.MockTransactionPool{}
		txPool.When("ValidateTransactionsForOrdering", mock.Any, mock.Any).Return(nil, errors.New("random error"))
		err := validateTxTransactionOrdering(ctx, cfg, txPool.ValidateTransactionsForOrdering, block.TransactionsBlock)
		require.Error(t, err)

		ok, err := txPool.Verify()
		require.True(t, ok)
		require.NoError(t, err)
	})
}

func TestConsensusContextValidateTransactionsBlockTriggerCompliance(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		tx := builders.TransferTransaction().Build()
		triggerTx := builders.TriggerTransaction().Build()
		tests := []struct {
			name           string
			triggerEnabled bool
			txs            []*protocol.SignedTransaction
			expectedToPass bool
			errorCause     error
		}{
			{
				"trigger disabled - empty block",
				false,
				[]*protocol.SignedTransaction{},
				true,
				nil,
			},
			{
				"trigger disabled - regular txs",
				false,
				[]*protocol.SignedTransaction{tx},
				true,
				nil,
			},
			{
				"trigger disabled - trigger at end",
				false,
				[]*protocol.SignedTransaction{tx, triggerTx},
				false,
				ErrTriggerDisabledAndTriggerExists,
			},
			{
				"trigger disabled - trigger not at end",
				false,
				[]*protocol.SignedTransaction{tx, triggerTx, tx},
				false,
				ErrTriggerDisabledAndTriggerExists,
			},
			{
				"trigger enabled - empty block",
				true,
				[]*protocol.SignedTransaction{},
				false,
				ErrTriggerEnabledAndTriggerMissing,
			},
			{
				"trigger enabled - only regular txs",
				true,
				[]*protocol.SignedTransaction{tx, tx},
				false,
				ErrTriggerEnabledAndTriggerMissing,
			},
			{
				"trigger enabled - trigger is not last",
				true,
				[]*protocol.SignedTransaction{triggerTx, tx},
				false,
				ErrTriggerEnabledAndTriggerMissing,
			},
			{
				"trigger enabled - more than one trigger",
				true,
				[]*protocol.SignedTransaction{triggerTx, tx, triggerTx},
				false,
				ErrTriggerEnabledAndTriggerNotLast,
			},
			{
				"trigger enabled - only one tx, is trigger",
				true,
				[]*protocol.SignedTransaction{triggerTx},
				true,
				nil,
			},
			{
				"trigger enabled - only one tx, is trigger",
				true,
				[]*protocol.SignedTransaction{triggerTx},
				true,
				nil,
			},
			{
				"trigger enabled - good block",
				true,
				[]*protocol.SignedTransaction{tx, tx, triggerTx},
				true,
				nil,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := config.ForConsensusContextTests(nil, tt.triggerEnabled)
				tb := &protocol.TransactionsBlockContainer{
					Header: (&protocol.TransactionsBlockHeaderBuilder{
						Timestamp: triggerTx.Transaction().Timestamp(),
					}).Build(),
					SignedTransactions: tt.txs,
				}
				err := validateTransactionsBlockTriggerCompliance(ctx, cfg, tb)
				if tt.expectedToPass {
					require.NoError(t, err, tt.name)
				} else {
					require.Error(t, err, tt.name)
					require.Equal(t, tt.errorCause, errors.Cause(err), "validation should fail error", err)
				}
			})
		}
	})
}
func TestConsensusContextValidateTransactionsBlockTriggerIsValid(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		cfg := config.ForConsensusContextTests(nil, false)
		tests := []struct {
			name           string
			tx             *protocol.SignedTransaction
			expectedToPass bool
		}{
			{
				"bad vcid",
				builders.TriggerTransaction().WithVirtualChainId(cfg.VirtualChainId() + 1).Build(),
				false,
			},
			{
				"bad protocol",
				builders.TriggerTransaction().WithProtocolVersion(cfg.ProtocolVersion() + 1).Build(),
				false,
			},
			{
				"has args",
				builders.TriggerTransaction().WithArgs(uint64(7)).Build(),
				false,
			},
			{
				"has signer",
				builders.TriggerTransaction().WithInvalidSignerScheme().Build(),
				false,
			},
			{
				"has signature",
				builders.Transaction().Build(),
				false,
			},
			{
				"good",
				builders.TriggerTransaction().Build(),
				true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				isOk := validateTransactionsBlockTxTriggerIsValid(tt.tx, cfg)
				require.Equal(t, tt.expectedToPass, isOk, "validator and expected don't match")
			})
		}
	})
}

func TestConsensusContextValidateTransactionsBlockTriggerIsValidTime(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		timestamp := time.Now()
		triggerTx := builders.TriggerTransaction().WithTimestamp(timestamp).Build()
		isOk := validateTransactionsBlockTxTriggerIsValidTime(triggerTx, primitives.TimestampNano(timestamp.UnixNano()))
		require.True(t, isOk, "validator time should find a all ok")
		isNotOk := validateTransactionsBlockTxTriggerIsValidTime(triggerTx, primitives.TimestampNano(timestamp.Add(time.Second).UnixNano()))
		require.False(t, isNotOk, "validator time should fail")
	})
}

func TestIsValidBlockTimestamp(t *testing.T) {
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
			jitter := 2 * time.Second
			currentBlockTimestamp := primitives.TimestampNano(now.Add(tt.currentBlockTimestampOffset).UnixNano())
			prevBlockTimestamp := primitives.TimestampNano(now.Add(tt.prevBlockTimestampOffset).UnixNano())
			err := isValidBlockTimestamp(currentBlockTimestamp, prevBlockTimestamp, primitives.TimestampNano(now.UnixNano()), primitives.TimestampNano(jitter.Nanoseconds()))
			if tt.expectedToPass {
				require.NoError(t, err, tt.name)
			} else {
				require.Error(t, err, tt.name)
			}
		})
	}
}
