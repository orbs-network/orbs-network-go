package leanhelixconsensus

import (
	"bytes"
	"context"
	"errors"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func validateBlockNotNil(block *protocol.BlockPairContainer) error {
	if block == nil || block.TransactionsBlock == nil || block.ResultsBlock == nil {
		return errors.New("BlockPair or either transactions or results block are nil")
	}
	return nil
}

func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash, prevBlock lh.Block) bool {
	// TODO Implement me

	// Validate Not Nil
	blockPair := FromLeanHelixBlock(block)

	if err := validateBlockNotNil(blockPair); err != nil {
		return false
	}

	// Validate Tx Block

	// Validate Rx Block

	// Validate Block Hash

	return true
}

func (p *blockProvider) ValidateBlockCommitment(blockHeight lhprimitives.BlockHeight, block lh.Block, blockHash lhprimitives.BlockHash) bool {

	blockPair := FromLeanHelixBlock(block)

	if err := validateBlockNotNil(blockPair); err != nil {
		return false
	}

	// *validate tx hash pointers*
	//Check the block's transactions_root_hash: Calculate the merkle root hash of the block's transactions and verify the hash in the header.
	txMerkleRoot := blockPair.TransactionsBlock.Header.RawTransactionsMerkleRootHash()
	if calculatedTxMerkleRoot, err := digest.CalcTransactionsMerkleRoot(blockPair.TransactionsBlock.SignedTransactions); err != nil {
		p.logger.Error("ValidateBlockCommitment error calculateTransactionsMerkleRoot")
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
