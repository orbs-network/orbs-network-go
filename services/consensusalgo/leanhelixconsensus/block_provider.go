package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

type BlockPairWrapper struct {
	blockPair *protocol.BlockPairContainer
}

func (b *BlockPairWrapper) Height() lhprimitives.BlockHeight {
	return lhprimitives.BlockHeight(b.blockPair.TransactionsBlock.Header.BlockHeight())
}

func ToLeanHelixBlock(blockPair *protocol.BlockPairContainer) leanhelix.Block {

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

func (p *blockProvider) ValidateBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, block leanhelix.Block, blockHash lhprimitives.BlockHash, prevBlock leanhelix.Block) bool {
	// TODO Implement me

	return true
}

func (p *blockProvider) ValidateBlockCommitment(blockHeight lhprimitives.BlockHeight, block leanhelix.Block, blockHash lhprimitives.BlockHash) bool {
	// TODO Implement me

	return true
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

func (p *blockProvider) RequestNewBlockProposal(ctx context.Context, blockHeight lhprimitives.BlockHeight, prevBlock leanhelix.Block) (leanhelix.Block, lhprimitives.BlockHash) {

	var newBlockHeight primitives.BlockHeight
	var prevTxBlockHash primitives.Sha256
	var prevRxBlockHash primitives.Sha256
	var prevBlockTimestamp primitives.TimestampNano

	if prevBlock == nil {
		newBlockHeight = 1
		prevTxBlockHash = nil
		prevRxBlockHash = nil
		prevBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano() - 1)

	} else {
		prevBlockWrapper := prevBlock.(*BlockPairWrapper)
		newBlockHeight = primitives.BlockHeight(prevBlock.Height() + 1)
		prevTxBlockHash = digest.CalcTransactionsBlockHash(prevBlockWrapper.blockPair.TransactionsBlock)
		prevRxBlockHash = digest.CalcResultsBlockHash(prevBlockWrapper.blockPair.ResultsBlock)
		prevBlockTimestamp = prevBlockWrapper.blockPair.TransactionsBlock.Header.Timestamp()
	}

	p.logger.Info("RequestNewBlockProposal()", log.Stringable("new-block-height", newBlockHeight))

	// TODO https://tree.taiga.io/project/orbs-network/us/642 Add configurable maxNumTx and maxBlockSize

	// get tx
	txOutput, err := p.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
		CurrentBlockHeight:      newBlockHeight,
		MaxBlockSizeKb:          0, // TODO(v1): fill in or remove from spec
		MaxNumberOfTransactions: 0,
		PrevBlockHash:           prevTxBlockHash,
		PrevBlockTimestamp:      prevBlockTimestamp,
	})
	if err != nil {
		return nil, nil
	}

	// get rx
	rxOutput, err := p.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
		CurrentBlockHeight: newBlockHeight,
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

// TODO (v1) Complete this https://tree.taiga.io/project/orbs-network/us/567
func (s *service) validateBlockConsensus(ctx context.Context, blockPair *protocol.BlockPairContainer, prevCommittedBlockPair *protocol.BlockPairContainer) error {
	// correct block type
	if !blockPair.TransactionsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.TransactionsBlock.BlockProof.Type())
	}
	if !blockPair.ResultsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.ResultsBlock.BlockProof.Type())
	}

	// TODO (v1) Impl in LH lib https://tree.taiga.io/project/orbs-network/us/473
	_ = s.leanHelix.ValidateBlockConsensus(ctx, ToLeanHelixBlock(blockPair), blockPair.TransactionsBlock.BlockProof.LeanHelix())
	return nil
}

func (p *blockProvider) GenerateGenesisBlock(ctx context.Context) *protocol.BlockPairContainer {
	return nil
}
