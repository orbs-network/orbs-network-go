// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	testValidators "github.com/orbs-network/orbs-network-go/test/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateBlockFailsOnNil(t *testing.T) {
	require.Error(t, validateBlockNotNil(nil, &validatorContext{}), "fail when BlockPair is nil")

	block := &protocol.BlockPairContainer{
		TransactionsBlock: nil,
		ResultsBlock:      &protocol.ResultsBlockContainer{},
	}
	require.Error(t, validateBlockNotNil(block, &validatorContext{}), "fail when transactions block is nil")
	block.TransactionsBlock = &protocol.TransactionsBlockContainer{}
	require.Nil(t, validateBlockNotNil(block, &validatorContext{}), "ok when blockPair's transaction and results blocks are not nil")
	block.ResultsBlock = nil
	require.Error(t, validateBlockNotNil(block, &validatorContext{}), "fail when results block is nil")
}

func TestValidateBlockCommitment_HappyFlow(t *testing.T) {
	block := testValidators.AStructurallyValidBlock()
	blockHash := []byte{1, 2, 3, 4}
	vcx := &validatorContext{
		blockHash:              primitives.Sha256(blockHash),
		CalcReceiptsMerkleRoot: digest.CalcReceiptsMerkleRoot,
		CalcStateDiffHash:      digest.CalcStateDiffHash,
	}

	require.True(t, validateBlockCommitmentInternal(1, ToLeanHelixBlock(block), blockHash, log.DefaultTestingLogger(t), vcx, &commitmentvalidators{
		validateBlockNotNil:                 func(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error { return nil },
		validateTransactionsBlockMerkleRoot: func(block *protocol.BlockPairContainer, vcx *validatorContext) error { return nil },
		validateTransactionsMetadataHash:    func(block *protocol.BlockPairContainer, vcx *validatorContext) error { return nil },
		validateReceiptsMerkleRoot:          func(block *protocol.BlockPairContainer, vcx *validatorContext) error { return nil },
		validateResultsBlockStateDiffHash:   func(block *protocol.BlockPairContainer, vcx *validatorContext) error { return nil },
		validateBlockHash_Commitment:        func(block *protocol.BlockPairContainer, vcx *validatorContext) error { return nil },
	}), "should return true when ValidateTransactionsBlock() and ValidateResultsBlock() are successful")
}
