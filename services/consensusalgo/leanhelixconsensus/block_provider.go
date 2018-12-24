package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
	"unsafe"
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
	consensusContext services.ConsensusContext,
	nodeAddress primitives.NodeAddress,
	nodePrivateKey primitives.EcdsaSecp256K1PrivateKey) *blockProvider {

	return &blockProvider{
		logger:           logger,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		nodeAddress:      nodeAddress,
		nodePrivateKey:   nodePrivateKey,
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

	p.logger.Info("RequestNewBlock()", log.Stringable("new-block-height", newBlockHeight))

	// get tx
	txOutput, err := p.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
		BlockHeight:        newBlockHeight,
		PrevBlockHash:      prevTxBlockHash,
		PrevBlockTimestamp: prevBlockTimestamp,
	})
	if err != nil {
		return nil, nil
	}

	// get rx
	rxOutput, err := p.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
		BlockHeight:       newBlockHeight,
		PrevBlockHash:     prevRxBlockHash,
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil, nil
	}

	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: txOutput.TransactionsBlock,
		ResultsBlock:      rxOutput.ResultsBlock,
	}

	// TODO: this seems hacky here - we should be able to transfer block without proof on the wire - currently the "empty" blockProof build was moved to consensusContext.create_block
	//blockPair.TransactionsBlock.BlockProof = (&protocol.TransactionsBlockProofBuilder{}).Build()
	//blockPair.ResultsBlock.BlockProof = (&protocol.ResultsBlockProofBuilder{}).Build()

	p.logger.Info("RequestNewBlock() returning", log.Int("num-transactions", len(txOutput.TransactionsBlock.SignedTransactions)), log.Int("num-receipts", len(rxOutput.ResultsBlock.TransactionReceipts)))

	blockHash := []byte(calculateBlockHash(blockPair))
	blockPairWrapper := ToLeanHelixBlock(blockPair)
	return blockPairWrapper, blockHash

}

// TODO Ask Oded/Gad is this is the correct impl! Oded said not to use XOR
func calculateBlockHash(blockPair *protocol.BlockPairContainer) primitives.Uint256 {
	txHash := digest.CalcTransactionsBlockHash(blockPair.TransactionsBlock)
	rxHash := digest.CalcResultsBlockHash(blockPair.ResultsBlock)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}

func sizeOfBlock(block *protocol.BlockPairContainer) int64 {
	txBlock := block.TransactionsBlock
	txBlockSize := len(txBlock.Header.Raw()) + len(txBlock.BlockProof.Raw()) + len(txBlock.Metadata.Raw())

	rsBlock := block.ResultsBlock
	rsBlockSize := len(rsBlock.Header.Raw()) + len(rsBlock.BlockProof.Raw())

	txBlockPointers := unsafe.Sizeof(txBlock) + unsafe.Sizeof(txBlock.Header) + unsafe.Sizeof(txBlock.Metadata) + unsafe.Sizeof(txBlock.BlockProof) + unsafe.Sizeof(txBlock.SignedTransactions)
	rsBlockPointers := unsafe.Sizeof(rsBlock) + unsafe.Sizeof(rsBlock.Header) + unsafe.Sizeof(rsBlock.BlockProof) + unsafe.Sizeof(rsBlock.TransactionReceipts) + unsafe.Sizeof(rsBlock.ContractStateDiffs)

	for _, tx := range txBlock.SignedTransactions {
		txBlockSize += len(tx.Raw())
		txBlockPointers += unsafe.Sizeof(tx)
	}
	for _, diff := range rsBlock.ContractStateDiffs {
		rsBlockSize += len(diff.Raw())
		rsBlockPointers += unsafe.Sizeof(diff)
	}
	for _, receipt := range rsBlock.TransactionReceipts {
		rsBlockSize += len(receipt.Raw())
		rsBlockPointers += unsafe.Sizeof(receipt)
	}
	pointers := unsafe.Sizeof(block) + txBlockPointers + rsBlockPointers

	return int64(txBlockSize) + int64(rsBlockSize) + int64(pointers)
}

// TODO Replace with code form lh-outline once finalized
func (s *service) validateBlockConsensus(blockPair *protocol.BlockPairContainer, prevCommittedBlockPair *protocol.BlockPairContainer) error {
	// correct block type
	if !blockPair.TransactionsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.TransactionsBlock.BlockProof.Type())
	}
	if !blockPair.ResultsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.ResultsBlock.BlockProof.Type())
	}

	// TODO Impl in LH lib https://tree.taiga.io/project/orbs-network/us/473
	_ = s.leanHelix.ValidateBlockConsensus(ToLeanHelixBlock(blockPair), nil, nil)
	return nil
}

// Get an empty block from consensusContext - it will also hold ProtocolVersion and VirtualChainID
// TODO: 2nd option is to pass virtual chain info to consensusAlgo
//  TODO: handle errors
func (p *blockProvider) GenerateGenesisBlock(ctx context.Context) *protocol.BlockPairContainer {

	//// get gensis tx
	//txOutput, err := p.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
	//	BlockHeight:             0,
	//	PrevBlockHash:           nil,
	//	PrevBlockTimestamp:      0, // Genesis block timeStamp will be ~ time.Now() (max(prev,now)+1)
	//	MaxNumberOfTransactions: 0,
	//})
	//if err != nil {
	//	panic(err)
	//}
	//
	//// get gensis rx
	//rxOutput, err := p.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
	//	BlockHeight:        0,
	//	PrevBlockHash:      nil,
	//	PrevBlockTimestamp: 0, // Genesis block timeStamp will be ~ time.Now()
	//	TransactionsBlock:  txOutput.TransactionsBlock,
	//})
	//if err != nil {
	//	panic(err)
	//}
	//
	//p.logger.Info(fmt.Sprintf("Genesis VirtualChainID: %d; ", txOutput.TransactionsBlock.Header.VirtualChainId()))
	//
	//blockPair := &protocol.BlockPairContainer{
	//	TransactionsBlock: txOutput.TransactionsBlock,
	//	ResultsBlock:      rxOutput.ResultsBlock,
	//}
	//
	//return blockPair

	return nil
}
