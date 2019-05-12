// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

type txValidator func(ctx context.Context, vctx *txValidatorContext) error

type txValidatorContext struct {
	protocolVersion        primitives.ProtocolVersion
	virtualChainId         primitives.VirtualChainId
	allowedTimestampJitter time.Duration
	input                  *services.ValidateTransactionsBlockInput
	txOrderValidator       func(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error)
}

func validateTxProtocolVersion(ctx context.Context, vctx *txValidatorContext) error {
	expectedProtocolVersion := vctx.protocolVersion
	checkedProtocolVersion := vctx.input.TransactionsBlock.Header.ProtocolVersion()
	if checkedProtocolVersion != expectedProtocolVersion {
		return errors.Wrapf(ErrMismatchedProtocolVersion, "expected %v actual %v", expectedProtocolVersion, checkedProtocolVersion)
	}
	return nil
}

func validateTxVirtualChainID(ctx context.Context, vctx *txValidatorContext) error {
	expectedVirtualChainId := vctx.virtualChainId
	checkedVirtualChainId := vctx.input.TransactionsBlock.Header.VirtualChainId()
	if checkedVirtualChainId != vctx.virtualChainId {
		return errors.Wrapf(ErrMismatchedVirtualChainID, "expected %v actual %v", expectedVirtualChainId, checkedVirtualChainId)
	}
	return nil
}

func validateTxBlockHeight(ctx context.Context, vctx *txValidatorContext) error {
	expectedBlockHeight := vctx.input.CurrentBlockHeight
	checkedBlockHeight := vctx.input.TransactionsBlock.Header.BlockHeight()
	if checkedBlockHeight != expectedBlockHeight {
		return ErrMismatchedBlockHeight
	}
	return nil
}

func validateTxPrevBlockHashPtr(ctx context.Context, vctx *txValidatorContext) error {
	expectedPrevBlockHashPtr := vctx.input.PrevBlockHash
	prevBlockHashPtr := vctx.input.TransactionsBlock.Header.PrevBlockHashPtr()
	if !bytes.Equal(prevBlockHashPtr, expectedPrevBlockHashPtr) {
		return errors.Wrapf(ErrMismatchedPrevBlockHash, "expected %v actual %v", expectedPrevBlockHashPtr, prevBlockHashPtr)
	}
	return nil
}

func validateTxTransactionsBlockTimestamp(ctx context.Context, vctx *txValidatorContext) error {
	prevBlockTimestamp := vctx.input.PrevBlockTimestamp
	currentBlockTimestamp := vctx.input.TransactionsBlock.Header.Timestamp()
	allowedTimestampJitter := vctx.allowedTimestampJitter
	now := time.Now()
	if err := isValidBlockTimestamp(currentBlockTimestamp, prevBlockTimestamp, now, allowedTimestampJitter); err != nil {
		return errors.Wrapf(ErrInvalidBlockTimestamp, "currentBlockTimestamp=%d prevBlockTimestamp=%d now=%s allowed_jitter=%s, err=%s",
			currentBlockTimestamp, prevBlockTimestamp, now, allowedTimestampJitter, err)
	}
	return nil
}

func validateTransactionsBlockMerkleRoot(ctx context.Context, vctx *txValidatorContext) error {
	return validators.ValidateTransactionsBlockMerkleRoot(&validators.BlockValidatorContext{
		TransactionsBlock: vctx.input.TransactionsBlock,
	})
}

func validateTransactionsBlockMetadataHash(ctx context.Context, vctx *txValidatorContext) error {
	return validators.ValidateTransactionsBlockMetadataHash(&validators.BlockValidatorContext{
		TransactionsBlock: vctx.input.TransactionsBlock,
	})
}

func validateTxTransactionOrdering(ctx context.Context, vctx *txValidatorContext) error {
	validationInput := &services.ValidateTransactionsForOrderingInput{
		CurrentBlockHeight:    vctx.input.TransactionsBlock.Header.BlockHeight(),
		CurrentBlockTimestamp: vctx.input.TransactionsBlock.Header.Timestamp(),
		SignedTransactions:    vctx.input.TransactionsBlock.SignedTransactions,
	}
	_, err := vctx.txOrderValidator(ctx, validationInput)
	if err != nil {
		return errors.Wrapf(ErrFailedTransactionOrdering, "%v", err)
	}
	return nil
}

func (s *service) ValidateTransactionsBlock(ctx context.Context, input *services.ValidateTransactionsBlockInput) (*services.ValidateTransactionsBlockOutput, error) {

	vctx := &txValidatorContext{
		protocolVersion:        s.config.ProtocolVersion(),
		virtualChainId:         s.config.VirtualChainId(),
		allowedTimestampJitter: s.config.ConsensusContextSystemTimestampAllowedJitter(),
		input:                  input,
		txOrderValidator:       s.transactionPool.ValidateTransactionsForOrdering,
	}

	validators := []txValidator{
		validateTxProtocolVersion,
		validateTxVirtualChainID,
		validateTxBlockHeight,
		validateTxPrevBlockHashPtr,
		validateTxTransactionsBlockTimestamp,
		validateTransactionsBlockMerkleRoot,
		validateTransactionsBlockMetadataHash,
		validateTxTransactionOrdering,
	}

	for _, v := range validators {
		if err := v(ctx, vctx); err != nil {
			return &services.ValidateTransactionsBlockOutput{}, err
		}
	}
	return &services.ValidateTransactionsBlockOutput{}, nil
}

func isValidBlockTimestamp(currentBlockTimestamp primitives.TimestampNano, prevBlockTimestamp primitives.TimestampNano, now time.Time, allowedTimestampJitter time.Duration) error {

	if allowedTimestampJitter < 0 {
		panic("allowedTimestampJitter cannot be negative")
	}

	upperJitterLimit := now.Add(allowedTimestampJitter)
	upperJitterLimitNano := upperJitterLimit.UnixNano()
	lowerJitterLimit := now.Add(-allowedTimestampJitter)
	lowerJitterLimitNano := lowerJitterLimit.UnixNano()

	prevBlockTimestampTime := time.Unix(0, int64(prevBlockTimestamp))
	currentBlockTimestampTime := time.Unix(0, int64(currentBlockTimestamp))

	if upperJitterLimitNano < 0 {
		panic("upperJitterLimit cannot be negative")
	}
	if lowerJitterLimitNano < 0 {
		panic("lowerJitterLimit cannot be negative")
	}

	if prevBlockTimestamp >= currentBlockTimestamp {
		return errors.Errorf("the previous block's timestamp is same or later than current block's timestamp: prevBlockTimestamp=%d (%s) currentBlockTimestamp=%d (%s)", prevBlockTimestamp, prevBlockTimestampTime, currentBlockTimestamp, currentBlockTimestampTime)
	}
	if uint64(currentBlockTimestamp) > uint64(upperJitterLimitNano) {
		return errors.Errorf("current block's timestamp is later than latest timestamp allowed (upper jitter limit): currentBlockTimestamp=%d (%s) upperJitterLimitNano=%d (%s)", uint64(currentBlockTimestamp), currentBlockTimestampTime, uint64(upperJitterLimitNano), upperJitterLimit)
	}

	if uint64(currentBlockTimestamp) < uint64(lowerJitterLimitNano) {
		return errors.Errorf("current block's timestamp is earlier than earliest timestamp allowed (lower jitter limit): currentBlockTimestamp=%d (%s) lowerJitterLimitNano=%d (%s)", uint64(currentBlockTimestamp), currentBlockTimestampTime, uint64(lowerJitterLimitNano), lowerJitterLimit)
	}
	return nil
}
