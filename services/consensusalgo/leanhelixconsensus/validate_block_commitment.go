package leanhelixconsensus

import (
	"bytes"
	"context"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"

	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

type blockValidator func(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error

type validatorContext struct {
	blockHash primitives.Sha256
}

type ValidateBlockFailsOnNilAdapter interface {
	ValidateBlockFailsOnNil(ctx context.Context, vc *validatorContext) error
}

func validateBlockNotNil(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error {
	if block == nil || block.TransactionsBlock == nil || block.ResultsBlock == nil {
		return errors.New("BlockPair or either transactions or results block are nil")
	}
	return nil
}

func validateTransactionsBlockMerkleRoot(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateTransactionsBlockMerkleRoot(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

func validateTransactionsMetadataHash(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateTransactionsBlockMetadataHash(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

func validateReceiptsMerkleRoot(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateReceiptsMerkleRoot(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

func validateResultsBlockStateDiffHash(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	return validators.ValidateResultsBlockStateDiffHash(&validators.BlockValidatorContext{
		TransactionsBlock: block.TransactionsBlock,
		ResultsBlock:      block.ResultsBlock,
	})
}

// TODO Consider moving to crypto/validators even though it's only used in this package
func validateBlockHash(block *protocol.BlockPairContainer, vcx *validatorContext) error {
	expectedBlockHash := vcx.blockHash
	calculatedBlockHash := []byte(digest.CalcBlockHash(block.TransactionsBlock, block.ResultsBlock))
	if !bytes.Equal(expectedBlockHash, calculatedBlockHash) {
		return errors.Wrapf(validators.ErrMismatchedBlockHash, "expected %v actual %v", expectedBlockHash, calculatedBlockHash)
	}
	return nil
}

func (p *blockProvider) ValidateBlockCommitment(blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash) bool {

	blockPair := FromLeanHelixBlock(block)

	validators := []blockValidator{
		validateBlockNotNil,
		validateTransactionsBlockMerkleRoot,
		validateTransactionsMetadataHash,
		validateReceiptsMerkleRoot,
		validateResultsBlockStateDiffHash,
		validateBlockHash,
	}

	vcx := &validatorContext{
		blockHash: primitives.Sha256(blockHash),
	}

	for _, validator := range validators {
		if err := validator(blockPair, vcx); err != nil {
			p.logger.Info("Error in ValidateBlockCommitment()")
			return false
		}
	}

	return true
}

/*
// validate deep hash - validate all inner hash pointers against given containers (merkleRoot
func (p *blockProvider) ValidateBlockCommitment(blockHeight lhprimitives.BlockHeight, block leanhelix.Block, blockHash lhprimitives.BlockHash) bool {

	blockPair := fromLeanHelixBlock(block)
	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil {
		p.logger.Error("ValidateBlockCommitment received an invalid block containing nil")
		return false
	}

	// *validate tx hash pointers*
	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
	txMerkleRoot := blockPair.TransactionsBlock.Header.RawTransactionsMerkleRootHash()
	if calculatedTxMerkleRoot, err := calculateTransactionsMerkleRoot(blockPair.TransactionsBlock.SignedTransactions); err != nil {
		p.logger.Error("ValidateBlockCommitment error calculateTransactionsMerkleRoot", log.Error(err))
		return false
	} else if !bytes.Equal(txMerkleRoot, []byte(calculatedTxMerkleRoot)) {
		p.logger.Error("ValidateBlockCommitment error transaction merkleRoot in header do not match txs")
		return false
	}

	//	Check the block's metadata hash: Calculate the hash of the block's metadata and verify the hash in the header.
	metadataHash := blockPair.TransactionsBlock.Header.RawMetadataHash()
	calculatedMetaDataHash := digest.CalcTransactionMetaDataHash(blockPair.TransactionsBlock.Metadata)
	if !bytes.Equal(metadataHash, []byte(calculatedMetaDataHash)) {
		p.logger.Error("ValidateBlockCommitment error transaction metadataHash in header do not match metadata")
		return false
	}

	// *validate rx hash pointers*
	//Check the block's receipts_root_hash: Calculate the merkle root hash of the block's receipts and verify the hash in the header.
	recieptsMerkleRoot := blockPair.ResultsBlock.Header.RawReceiptsMerkleRootHash()
	if calculatedRecieptMerkleRoot, err := calculateReceiptsMerkleRoot(blockPair.ResultsBlock.TransactionReceipts); err != nil {
		p.logger.Error("ValidateBlockCommitment error calculateReceiptsMerkleRoot", log.Error(err))
		return false
	} else if !bytes.Equal(recieptsMerkleRoot, []byte(calculatedRecieptMerkleRoot)) {
		p.logger.Error("ValidateBlockCommitment error receipt merkleRoot in header do not match txs receipts")
		return false
	}

	//Check the block's state_diff_hash: Calculate the hash of the block's state diff and verify the hash in the header.
	stateDiffMerkleRoot := blockPair.ResultsBlock.Header.RawStateDiffHash()
	if calculatedStateDiffMerkleRoot, err := calculateStateDiffMerkleRoot(blockPair.ResultsBlock.ContractStateDiffs); err != nil {
		p.logger.Error("ValidateBlockCommitment error calculateStateDiffMerkleRoot", log.Error(err))
		return false
	} else if !bytes.Equal(stateDiffMerkleRoot, []byte(calculatedStateDiffMerkleRoot)) {
		p.logger.Error("ValidateBlockCommitment error state diff merkleRoot in header do not match state diffs")
		return false
	}

	// validate blockHash
	calcBlockHash := []byte(digest.CalcBlockHash(blockPair.TransactionsBlock, blockPair.ResultsBlock))
	if !bytes.Equal([]byte(blockHash), calcBlockHash) {
		p.logger.Error("ValidateBlockCommitment blockHash mismatch")
		return false
	}

	return true
}


*/
