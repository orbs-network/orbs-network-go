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

func FromLeanHelixBlock(lhBlock lh.Block) *protocol.BlockPairContainer {
	if lhBlock != nil {
		block, ok := lhBlock.(*BlockPairWrapper)
		if ok {
			return block.blockPair
		}
	}
	return nil
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
	var prevTxBlockHash primitives.Sha256
	var prevRxBlockHash primitives.Sha256
	var prevBlockTimestamp primitives.TimestampNano

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

func (s *service) validateBlockConsensus(ctx context.Context, blockPair *protocol.BlockPairContainer, prevBlockPair *protocol.BlockPairContainer) error {

	if err := validLeanHelixBlockPair(blockPair); err != nil {
		return err
	}

	if err := validLeanHelixBlockPair(prevBlockPair); err != nil {
		return err
	}

	blockProof := blockPair.TransactionsBlock.BlockProof.LeanHelix()
	prevBlockProof := prevBlockPair.TransactionsBlock.BlockProof.LeanHelix()

	err := s.leanHelix.ValidateBlockConsensus(ctx, ToLeanHelixBlock(blockPair), blockProof, prevBlockProof)
	if err != nil {
		return errors.Wrap(err, "LeanHelix: ValidateBlockConsensus() invalid blockProof")
	}
	return nil
}

func validLeanHelixBlockPair(blockPair *protocol.BlockPairContainer) error {
	if blockPair == nil {
		return errors.New("LeanHelix: nil blockPair")
	}
	if blockPair.TransactionsBlock == nil {
		return errors.New("LeanHelix: nil Transactions Block")
	}
	if blockPair.ResultsBlock == nil {
		return errors.New("LeanHelix: nil Results Block")
	}
	if blockPair.TransactionsBlock.BlockProof == nil {
		return errors.New("LeanHelix: nil transactions block proof")
	}
	if blockPair.ResultsBlock.BlockProof == nil {
		return errors.New("LeanHelix: nil results block proof")
	}
	// correct block type
	if !blockPair.TransactionsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("LeanHelix: incorrect block proof type for transaction block: %v", blockPair.TransactionsBlock.BlockProof.Type())
	}
	if !blockPair.ResultsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("LeanHelix: incorrect block proof type for results block: %v", blockPair.ResultsBlock.BlockProof.Type())
	}
	// same block proof in txBlock and rxBlock
	if !bytes.Equal(blockPair.TransactionsBlock.BlockProof.LeanHelix(), blockPair.ResultsBlock.BlockProof.LeanHelix()) {
		return errors.Errorf("LeanHelix: TransactionsBlock LeanHelix block proof and  ResultsBlock LeanHelix block proof do not match")
	}
	return nil
}

// Genesis is defined to be nil
func (p *blockProvider) GenerateGenesisBlockProposal(ctx context.Context) (lh.Block, lhprimitives.BlockHash) {
	return nil, nil
}
