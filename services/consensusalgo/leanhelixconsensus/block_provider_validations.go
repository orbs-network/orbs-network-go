package leanhelixconsensus

import (
	"bytes"
	"context"
	"github.com/pkg/errors"

	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/validators"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type blockValidator func(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error

type validatorContext struct {
	blockHash primitives.Sha256
}

type ValidateBlockFailsOnNilAdapter interface {
	ValidateBlockFailsOnNil(ctx context.Context, vc *validatorContext) error
}

type realValidateBlockFailsOnNilAdapter struct {
	validateBlockFailsOnNil func(ctx context.Context, vc *validatorContext) error
}

func (r *realValidateBlockFailsOnNilAdapter) ValidateBlockFailsOnNil(ctx context.Context, vc *validatorContext) error {
	return r.validateBlockFailsOnNil(ctx, vc)
}
func NewRealValidateBlockFailsOnNilAdapter(f func(ctx context.Context, vc *validatorContext) error) ValidateBlockFailsOnNilAdapter {
	return &realValidateBlockFailsOnNilAdapter{
		validateBlockFailsOnNil: f,
	}
}

func validateBlockNotNil(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error {
	if block == nil || block.TransactionsBlock == nil || block.ResultsBlock == nil {
		return errors.New("BlockPair or either transactions or results block are nil")
	}
	return nil
}

func validateTxBlock(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error {
	return nil
}

func validateRxBlock(block *protocol.BlockPairContainer, validatorCtx *validatorContext) error {
	return nil
}

func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash, prevBlock lh.Block) bool {
	// TODO Implement me

	// Validate Not Nil
	blockPair := FromLeanHelixBlock(block)

	validators := []blockValidator{
		validateBlockNotNil,
	}

	vcx := &validatorContext{}

	for _, validator := range validators {
		if err := validator(blockPair, vcx); err != nil {
			p.logger.Info("Error in ValidateBlockProposal()", log.Error(err))
			return false
		}
	}

	newBlockHeight := primitives.BlockHeight(1)
	var prevTxBlockHash primitives.Sha256 = nil
	var prevRxBlockHash primitives.Sha256 = nil
	//var prevBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) - 1
	var prevBlockTimestamp primitives.TimestampNano = 0

	if prevBlock != nil {
		prevBlockPair := FromLeanHelixBlock(prevBlock)
		newBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlock := prevBlockPair.TransactionsBlock
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevTxBlock)
		prevBlockTimestamp = prevTxBlock.Header.Timestamp()
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockPair.ResultsBlock)
	}

	// Validate Tx Block
	// also inner hashPointers are checked
	_, err := p.consensusContext.ValidateTransactionsBlock(ctx, &services.ValidateTransactionsBlockInput{
		CurrentBlockHeight: newBlockHeight,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockHash:      prevTxBlockHash,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateTransactionsBlock", log.Error(err))
		return false
	}

	// Validate Rx Block
	// also inner hashPointers are checked
	_, err = p.consensusContext.ValidateResultsBlock(ctx, &services.ValidateResultsBlockInput{
		CurrentBlockHeight: newBlockHeight,
		ResultsBlock:       blockPair.ResultsBlock,
		PrevBlockHash:      prevRxBlockHash,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateResultsBlock", log.Int("block-height", int(newBlockHeight)), log.Error(err))
		return false
	}

	// validate blockHash
	calcBlockHash := []byte(digest.CalcBlockHash(blockPair.TransactionsBlock, blockPair.ResultsBlock))
	if !bytes.Equal([]byte(blockHash), calcBlockHash) {
		p.logger.Error("ValidateBlockProposal blockHash mismatch")
		return false
	}
	p.logger.Info("ValidateBlockProposal passed", log.Int("block-height", int(newBlockHeight)))
	return true
}

//func validateTxTransactionsBlockMerkleRoot(ctx context.Context, vctx *txValidatorContext) error {
//	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
//	txMerkleRoot := vctx.input.TransactionsBlock.Header.TransactionsMerkleRootHash()
//	if expectedTxMerkleRoot, err := digest.CalcTransactionsMerkleRoot(vctx.input.TransactionsBlock.SignedTransactions); err != nil {
//		return err
//	} else if !bytes.Equal(txMerkleRoot, expectedTxMerkleRoot) {
//		return errors.Wrapf(ErrMismatchedTxMerkleRoot, "expected %v actual %v", expectedTxMerkleRoot, txMerkleRoot)
//	}
//	return nil
//}
//
//func validateTxMetadataHash(ctx context.Context, vctx *txValidatorContext) error {
//	//	Check the block's metadata hash: Calculate the hash of the block's metadata and verify the hash in the header.
//	expectedMetaDataHash := digest.CalcTransactionMetaDataHash(vctx.input.TransactionsBlock.Metadata)
//	metadataHash := vctx.input.TransactionsBlock.Header.MetadataHash()
//	if !bytes.Equal(metadataHash, expectedMetaDataHash) {
//		return errors.Wrapf(ErrMismatchedMetadataHash, "expected %v actual %v", expectedMetaDataHash, metadataHash)
//	}
//	return nil
//}

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
			p.logger.Info("Error in ValidateBlockCommitment()", log.Error(err))
			return false
		}
	}

	// validate blockHash

	return true
}

// TODO Remove this reference code when done digesting it
/*

// Called to  validate leader block proposal.
// Full block validation - content (block_height, prevBlock pointer, timeStamp, protocolVersion, virtualChainId - using consensusContext)
// and structure (inner hashPointers and blockHash).
// Assume prevBlock is valid and under consensus.
// Assume consensusContext validation also checks inner hashPointers
func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block leanhelix.Block, blockHash lhprimitives.BlockHash, prevBlock leanhelix.Block) bool {
	blockPair := fromLeanHelixBlock(block)
	if blockPair == nil || blockPair.TransactionsBlock == nil || blockPair.ResultsBlock == nil {
		p.logger.Error("ValidateBlockProposal received an invalid block containing nil")
		return false
	}

	newBlockHeight := primitives.BlockHeight(1)
	var prevTxBlockHash primitives.Sha256 = nil
	var prevRxBlockHash primitives.Sha256 = nil
	//var prevBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) - 1
	var prevBlockTimestamp primitives.TimestampNano = 0

	if prevBlock != nil {
		prevBlockPair := fromLeanHelixBlock(prevBlock)
		newBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlock := prevBlockPair.TransactionsBlock
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevTxBlock)
		prevBlockTimestamp = prevTxBlock.Header.Timestamp()
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockPair.ResultsBlock)
	}

	// also inner hashPointers are checked
	_, err := p.consensusContext.ValidateTransactionsBlock(ctx, &services.ValidateTransactionsBlockInput{
		BlockHeight:        newBlockHeight,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockHash:      prevTxBlockHash,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateTransactionsBlock", log.Error(err))
		return false
	}

	// also inner hashPointers are checked
	_, err = p.consensusContext.ValidateResultsBlock(ctx, &services.ValidateResultsBlockInput{
		BlockHeight:        newBlockHeight,
		ResultsBlock:       blockPair.ResultsBlock,
		PrevBlockHash:      prevRxBlockHash,
		TransactionsBlock:  blockPair.TransactionsBlock,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		p.logger.Error("ValidateBlockProposal failed ValidateResultsBlock", log.Int("block-height", int(newBlockHeight)), log.Error(err))
		return false
	}

	// validate blockHash
	calcBlockHash := []byte(digest.CalcBlockHash(blockPair.TransactionsBlock, blockPair.ResultsBlock))
	if !bytes.Equal([]byte(blockHash), calcBlockHash) {
		p.logger.Error("ValidateBlockProposal blockHash mismatch")
		return false
	}
	p.logger.Info("ValidateBlockProposal passed", log.Int("block-height", int(newBlockHeight)))
	return true
}


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
