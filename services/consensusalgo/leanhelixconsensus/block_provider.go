package leanhelixconsensus

import (
	"bytes"
	"context"
	"github.com/orbs-network/lean-helix-go"
	lh "github.com/orbs-network/lean-helix-go/services/interfaces"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

type BlockPairWrapper struct {
	blockPair *protocol.BlockPairContainer
}

func (b *BlockPairWrapper) Height() lhprimitives.BlockHeight {
	return lhprimitives.BlockHeight(b.blockPair.TransactionsBlock.Header.BlockHeight())
}

func ToLeanHelixBlock(blockPair *protocol.BlockPairContainer) lh.Block {

	if blockPair == nil {
		return nil
	}
	return &BlockPairWrapper{
		blockPair: blockPair,
	}
}

type blockProvider struct {
	logger           log.BasicLogger
	leanhelix        leanhelix.LeanHelix
	blockStorage     services.BlockStorage
	consensusContext services.ConsensusContext
	nodeAddress      primitives.NodeAddress
	nodePrivateKey   primitives.EcdsaSecp256K1PrivateKey
}

func NewBlockProvider(
	logger log.BasicLogger,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext) *blockProvider {

	return &blockProvider{
		logger:           logger,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
	}

}

func (p *blockProvider) RequestNewBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, prevBlock lh.Block) (lh.Block, lhprimitives.BlockHash) {

	currentBlockHeight := primitives.BlockHeight(1)
	var prevTxBlockHash primitives.Sha256 = nil
	var prevRxBlockHash primitives.Sha256 = nil
	var prevBlockTimestamp primitives.TimestampNano = 0

	if prevBlock != nil {
		prevBlockWrapper := prevBlock.(*BlockPairWrapper)
		currentBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevBlockWrapper.blockPair.TransactionsBlock)
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockWrapper.blockPair.ResultsBlock)
		prevBlockTimestamp = prevBlockWrapper.blockPair.TransactionsBlock.Header.Timestamp()
	}

	p.logger.Info("RequestNewBlockProposal()", log.Stringable("new-block-height", currentBlockHeight))

	// TODO https://tree.taiga.io/project/orbs-network/us/642 Add configurable maxNumTx and maxBlockSize
	maxNumOfTransactions := uint32(10000)
	maxBlockSize := uint32(1000000)

	// get tx
	txOutput, err := p.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
		CurrentBlockHeight:      currentBlockHeight,
		PrevBlockHash:           prevTxBlockHash,
		PrevBlockTimestamp:      prevBlockTimestamp,
		MaxNumberOfTransactions: maxNumOfTransactions,
		MaxBlockSizeKb:          maxBlockSize,
	})
	if err != nil {
		return nil, nil
	}

	// get rx
	rxOutput, err := p.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
		CurrentBlockHeight: currentBlockHeight,
		PrevBlockHash:      prevRxBlockHash,
		TransactionsBlock:  txOutput.TransactionsBlock,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		return nil, nil
	}

	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: txOutput.TransactionsBlock,
		ResultsBlock:      rxOutput.ResultsBlock,
	}

	p.logger.Info("RequestNewBlockProposal() returning", log.Int("num-transactions", len(txOutput.TransactionsBlock.SignedTransactions)), log.Int("num-receipts", len(rxOutput.ResultsBlock.TransactionReceipts)))

	blockHash := []byte(digest.CalcBlockHash(blockPair.TransactionsBlock, blockPair.ResultsBlock))
	blockPairWrapper := ToLeanHelixBlock(blockPair)
	return blockPairWrapper, blockHash

}

// TODO (v1) Complete this including unit tests, see: https://tree.taiga.io/project/orbs-network/us/567
func (s *service) validateBlockConsensus(ctx context.Context, blockPair *protocol.BlockPairContainer, prevBlockPair *protocol.BlockPairContainer) error {
	if blockPair.TransactionsBlock.BlockProof == nil || blockPair.ResultsBlock.BlockProof == nil {
		return errors.New("nil block proof")
	}
	// correct block type
	if !blockPair.TransactionsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type for transaction block: %v", blockPair.TransactionsBlock.BlockProof.Type())
	}
	if !blockPair.ResultsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type for results block: %v", blockPair.ResultsBlock.BlockProof.Type())
	}

	// same block proof in txBlock and rxBlock
	if !bytes.Equal(blockPair.TransactionsBlock.BlockProof.LeanHelix(), blockPair.ResultsBlock.BlockProof.LeanHelix()) {
		return errors.Errorf("TransactionsBlock LeanHelix block proof and  ResultsBlock LeanHelix block proof do not match")
	}

	// TODO (v1) Impl in LH lib https://tree.taiga.io/project/orbs-network/us/473
	isBlockProofValid := true //isBlockProofValid := s.leanHelix.ValidateBlockConsensus(ctx, ToLeanHelixBlock(blockPair), blockPair.TransactionsBlock.BlockProof.LeanHelix(), prevBlockPair.TransactionsBlock.BlockProof.LeanHelix())
	if !isBlockProofValid {
		return errors.Errorf("LeanHelix ValidateBlockConsensus - block proof is not valid!!")
	}
	return nil
}

// Genesis is defined to be nil
func (p *blockProvider) GenerateGenesisBlockProposal(ctx context.Context) (lh.Block, lhprimitives.BlockHash) {
	return nil, nil
}
