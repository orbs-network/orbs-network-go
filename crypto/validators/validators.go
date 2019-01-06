package validators

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type BlockValidatorContext struct {
	TransactionsBlock       *protocol.TransactionsBlockContainer
	ResultsBlock            *protocol.ResultsBlockContainer
	CalcReceiptsMerkleRoot  func(receipts []*protocol.TransactionReceipt) (primitives.Sha256, error)
	CalcStateDiffMerkleRoot func(stateDiffs []*protocol.ContractStateDiff) (primitives.Sha256, error)
}

var ErrMismatchedTxMerkleRoot = errors.New("mismatched transactions merkle root")
var ErrMismatchedMetadataHash = errors.New("mismatched metadata hash")
var ErrMismatchedReceiptsRootHash = errors.New("receipt merkleRoot is different between results block header and calculated transaction receipts")
var ErrCalcReceiptsMerkleRoot = errors.New("failed in CalcReceiptsMerkleRoot()")
var ErrMismatchedStateDiffHash = errors.New("state diff merkleRoot is different between results block header and calculated transaction receipts")
var ErrCalcStateDiffMerkleRoot = errors.New("failed in ErrCalcStateDiffMerkleRoot()")
var ErrMismatchedBlockHash = errors.New("mismatched calculated block hash")

func ValidateTransactionsBlockMerkleRoot(bvcx *BlockValidatorContext) error {
	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
	transactionsMerkleRoot := bvcx.TransactionsBlock.Header.TransactionsMerkleRootHash()
	if expectedTransactionsMerkleRoot, err := digest.CalcTransactionsMerkleRoot(bvcx.TransactionsBlock.SignedTransactions); err != nil {
		return err
	} else if !bytes.Equal(transactionsMerkleRoot, expectedTransactionsMerkleRoot) {
		return errors.Wrapf(ErrMismatchedTxMerkleRoot, "expected %v actual %v", expectedTransactionsMerkleRoot, transactionsMerkleRoot)
	}
	return nil
}

func ValidateTransactionsBlockMetadataHash(bvcx *BlockValidatorContext) error {
	//	Check the block's metadata hash: Calculate the hash of the block's metadata and verify the hash in the header.
	expectedMetaDataHash := digest.CalcTransactionMetaDataHash(bvcx.TransactionsBlock.Metadata)
	metadataHash := bvcx.TransactionsBlock.Header.MetadataHash()
	if !bytes.Equal(metadataHash, expectedMetaDataHash) {
		return errors.Wrapf(ErrMismatchedMetadataHash, "expected %v actual %v", expectedMetaDataHash, metadataHash)
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
		return errors.Wrapf(ErrMismatchedReceiptsRootHash, "expected %v actual %v", expectedReceiptsMerkleRoot, calculatedReceiptMerkleRoot)
	}
	return nil
}

func ValidateResultsBlockStateDiffHash(bvcx *BlockValidatorContext) error {
	expectedStateDiffMerkleRoot := bvcx.ResultsBlock.Header.StateDiffHash()
	calculatedStateDiffMerkleRoot, err := bvcx.CalcStateDiffMerkleRoot(bvcx.ResultsBlock.ContractStateDiffs)
	if err != nil {
		return errors.Wrapf(ErrCalcStateDiffMerkleRoot, "ValidateResultsBlock error calculateStateDiffMerkleRoot(), %v", err)
	}
	if !bytes.Equal(expectedStateDiffMerkleRoot, []byte(calculatedStateDiffMerkleRoot)) {
		return errors.Wrapf(ErrMismatchedStateDiffHash, "expected %v actual %v", expectedStateDiffMerkleRoot, calculatedStateDiffMerkleRoot)
	}
	return nil
}
