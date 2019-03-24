// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package validators

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type BlockValidatorContext struct {
	TransactionsBlock      *protocol.TransactionsBlockContainer
	ResultsBlock           *protocol.ResultsBlockContainer
	CalcReceiptsMerkleRoot func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
	CalcStateDiffHash      func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
	ExpectedBlockHash      primitives.Sha256
}

var ErrMismatchedTxMerkleRoot = errors.New("ErrMismatchedTxMerkleRoot mismatched transactions merkle root")
var ErrMismatchedMetadataHash = errors.New("ErrMismatchedMetadataHash mismatched metadata hash")
var ErrMismatchedReceiptsRootHash = errors.New("ErrMismatchedReceiptsRootHash receipt merkleRoot is different between results block header and calculated transaction receipts")
var ErrCalcReceiptsMerkleRoot = errors.New("ErrCalcReceiptsMerkleRoot failed in CalcReceiptsMerkleRoot()")
var ErrMismatchedStateDiffHash = errors.New("ErrMismatchedStateDiffHash state diff merkleRoot is different between results block header and calculated transaction receipts")
var ErrCalcStateDiffHash = errors.New("ErrCalcStateDiffHash failed in ErrCalcStateDiffHash()")
var ErrMismatchedBlockHash = errors.New("ErrMismatchedBlockHash mismatched calculated block hash")

func ValidateTransactionsBlockMerkleRoot(bvcx *BlockValidatorContext) error {
	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
	transactionsMerkleRoot := bvcx.TransactionsBlock.Header.TransactionsMerkleRootHash()
	if expectedTransactionsMerkleRoot, err := digest.CalcTransactionsMerkleRoot(bvcx.TransactionsBlock.SignedTransactions); err != nil {
		return err
	} else if !bytes.Equal(transactionsMerkleRoot, expectedTransactionsMerkleRoot) {
		return errors.Wrapf(ErrMismatchedTxMerkleRoot, "expected=%v actual=%v", expectedTransactionsMerkleRoot, transactionsMerkleRoot)
	}
	return nil
}

func ValidateTransactionsBlockMetadataHash(bvcx *BlockValidatorContext) error {
	//	Check the block's metadata hash: Calculate the hash of the block's metadata and verify the hash in the header.
	expectedMetaDataHash := digest.CalcTransactionMetaDataHash(bvcx.TransactionsBlock.Metadata)
	metadataHash := bvcx.TransactionsBlock.Header.MetadataHash()
	if !bytes.Equal(metadataHash, expectedMetaDataHash) {
		return errors.Wrapf(ErrMismatchedMetadataHash, "expected=%v actual=%v", expectedMetaDataHash, metadataHash)
	}
	return nil
}

func ValidateReceiptsMerkleRoot(bvcx *BlockValidatorContext) error {
	expectedReceiptsMerkleRoot := bvcx.ResultsBlock.Header.ReceiptsMerkleRootHash()
	calculatedReceiptMerkleRoot, err := bvcx.CalcReceiptsMerkleRoot(bvcx.ResultsBlock.TransactionReceipts)
	if err != nil {
		return errors.Wrapf(ErrCalcReceiptsMerkleRoot, "ValidateResultsBlock error calculateReceiptsMerkleRoot(), %v", err)
	}
	if !bytes.Equal(expectedReceiptsMerkleRoot, []byte(calculatedReceiptMerkleRoot)) {
		return errors.Wrapf(ErrMismatchedReceiptsRootHash, "expected=%v actual=%v", expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot)
	}
	return nil
}

func ValidateResultsBlockStateDiffHash(bvcx *BlockValidatorContext) error {
	expectedStateDiffHash := bvcx.ResultsBlock.Header.StateDiffHash()
	calculatedStateDiffHash, err := bvcx.CalcStateDiffHash(bvcx.ResultsBlock.ContractStateDiffs)
	if err != nil {
		return errors.Wrapf(ErrCalcStateDiffHash, "ValidateResultsBlock error calculateStateDiffHash(), %v", err)
	}
	if !bytes.Equal(expectedStateDiffHash, []byte(calculatedStateDiffHash)) {
		return errors.Wrapf(ErrMismatchedStateDiffHash, "expected=%v actual=%v", expectedStateDiffHash, calculatedStateDiffHash)
	}
	return nil
}

func ValidateBlockHash(bvcx *BlockValidatorContext) error {
	if bvcx.TransactionsBlock == nil || bvcx.ResultsBlock == nil {
		return errors.New("nil block")
	}
	calculatedBlockHash := []byte(digest.CalcBlockHash(bvcx.TransactionsBlock, bvcx.ResultsBlock))
	if !bytes.Equal(bvcx.ExpectedBlockHash, calculatedBlockHash) {
		return errors.Wrapf(ErrMismatchedBlockHash, "expected=%v actual=%v", bvcx.ExpectedBlockHash, calculatedBlockHash)
	}
	return nil
}
