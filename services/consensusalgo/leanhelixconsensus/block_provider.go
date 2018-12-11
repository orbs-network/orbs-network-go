package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

type BlockPairWrapper struct {
	blockPair *protocol.BlockPairContainer
}

func (b *BlockPairWrapper) Height() lhprimitives.BlockHeight {
	return lhprimitives.BlockHeight(b.blockPair.TransactionsBlock.Header.BlockHeight())
}

func ToBlockPairWrapper(blockPair *protocol.BlockPairContainer) *BlockPairWrapper {
	return &BlockPairWrapper{
		blockPair: blockPair,
	}
}

type blockProvider struct {
	logger           log.BasicLogger
	blockStorage     services.BlockStorage
	consensusContext services.ConsensusContext
	nodePublicKey    primitives.Ed25519PublicKey
	nodePrivateKey   primitives.Ed25519PrivateKey
}

func NewBlockProvider(
	logger log.BasicLogger,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey) *blockProvider {

	return &blockProvider{
		logger:           logger,
		blockStorage:     blockStorage,
		consensusContext: consensusContext,
		nodePublicKey:    nodePublicKey,
		nodePrivateKey:   nodePrivateKey,
	}

}

func (p *blockProvider) RequestNewBlock(ctx context.Context, prevBlock leanhelix.Block) leanhelix.Block {
	blockWrapper := prevBlock.(*BlockPairWrapper)

	newBlockHeight := primitives.BlockHeight(prevBlock.Height() + 1)

	p.logger.Info("RequestNewBlock()", log.Stringable("new-block-height", newBlockHeight))

	// get tx
	txOutput, err := p.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
		BlockHeight:        newBlockHeight,
		PrevBlockHash:      digest.CalcTransactionsBlockHash(blockWrapper.blockPair.TransactionsBlock),
		PrevBlockTimestamp: blockWrapper.blockPair.TransactionsBlock.Header.Timestamp(),
	})
	if err != nil {
		return nil
	}

	// get rx
	rxOutput, err := p.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
		BlockHeight:       newBlockHeight,
		PrevBlockHash:     digest.CalcResultsBlockHash(blockWrapper.blockPair.ResultsBlock),
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil
	}

	// generate signed block
	pair, err := signBlockProposal(txOutput.TransactionsBlock, rxOutput.ResultsBlock, p.nodePrivateKey)
	blockPairWrapper := ToBlockPairWrapper(pair)
	if err != nil {
		return nil
	}

	p.logger.Info("RequestNewBlock() returning", log.Int("num-transactions", len(txOutput.TransactionsBlock.SignedTransactions)), log.Int("num-receipts", len(rxOutput.ResultsBlock.TransactionReceipts)))

	return blockPairWrapper

}

func (p *blockProvider) CalculateBlockHash(block leanhelix.Block) lhprimitives.Uint256 {
	blockPairWrapper, ok := block.(*BlockPairWrapper)
	if !ok {
		return nil
	}
	return deepHash(blockPairWrapper.blockPair.TransactionsBlock, blockPairWrapper.blockPair.ResultsBlock)
}

func deepHash(txBlock *protocol.TransactionsBlockContainer, rxBlock *protocol.ResultsBlockContainer) []byte {
	txHash := digest.CalcTransactionsBlockHash(txBlock)
	rxHash := digest.CalcResultsBlockHash(rxBlock)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}

func (p *blockProvider) ValidateBlock(block leanhelix.Block) bool {
	if block == nil {
		return false
	}
	blockWrapper, ok := block.(*BlockPairWrapper)
	if !ok {
		return false
	}
	if blockWrapper.blockPair == nil {
		return false
	}
	if blockWrapper.blockPair.TransactionsBlock == nil || blockWrapper.blockPair.ResultsBlock == nil {
		return false
	}
	if blockWrapper.blockPair.TransactionsBlock.Header == nil {
		return false
	}
	return true
}

func generateGenesisBlock(nodePrivateKey primitives.Ed25519PrivateKey) *protocol.BlockPairContainer {
	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header:             (&protocol.TransactionsBlockHeaderBuilder{BlockHeight: 0}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{},
		BlockProof:         nil, // will be generated in a minute when signed
	}
	resultsBlock := &protocol.ResultsBlockContainer{
		Header:                  (&protocol.ResultsBlockHeaderBuilder{BlockHeight: 0}).Build(),
		TransactionsBloomFilter: (&protocol.TransactionsBloomFilterBuilder{}).Build(),
		TransactionReceipts:     []*protocol.TransactionReceipt{},
		ContractStateDiffs:      []*protocol.ContractStateDiff{},
		BlockProof:              nil, // will be generated in a minute when signed
	}
	blockPair, err := signBlockProposal(transactionsBlock, resultsBlock, nodePrivateKey)
	if err != nil {
		//s.logger.Error("leader failed to sign genesis block", log.Error(err))
		return nil
	}
	return blockPair
}

func (s *service) validateBlockConsensus(blockPair *protocol.BlockPairContainer, prevCommittedBlockPair *protocol.BlockPairContainer) error {
	// correct block type
	if !blockPair.TransactionsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.TransactionsBlock.BlockProof.Type())
	}
	if !blockPair.ResultsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.ResultsBlock.BlockProof.Type())
	}

	// TODO Impl in LH lib https://tree.taiga.io/project/orbs-network/us/473
	return nil
}

func signBlockProposal(transactionsBlock *protocol.TransactionsBlockContainer, resultsBlock *protocol.ResultsBlockContainer, nodePrivateKey primitives.Ed25519PrivateKey) (*protocol.BlockPairContainer, error) {
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock:      resultsBlock,
	}

	// prepare signature over the block headers
	blockPairDataToSign := dataToSignFrom(blockPair)
	_, err := signature.SignEd25519(nodePrivateKey, blockPairDataToSign)
	if err != nil {
		return nil, err
	}

	// generate tx block proof
	blockPair.TransactionsBlock.BlockProof = (&protocol.TransactionsBlockProofBuilder{
		Type:      protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: &consensus.LeanHelixBlockProofBuilder{
			// TODO Transactions BlockProof goes here https://tree.taiga.io/project/orbs-network/us/529
			// See https://tree.taiga.io/project/orbs-network/us/529
		},
	}).Build()

	// generate rx block proof
	blockPair.ResultsBlock.BlockProof = (&protocol.ResultsBlockProofBuilder{
		Type:      protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: &consensus.LeanHelixBlockProofBuilder{
			// TODO Results BlockProof goes here https://tree.taiga.io/project/orbs-network/us/529
			// See https://tree.taiga.io/project/orbs-network/us/529
		},
	}).Build()
	return blockPair, nil
}

func dataToSignFrom(blockPair *protocol.BlockPairContainer) []byte {
	return deepHash(blockPair.TransactionsBlock, blockPair.ResultsBlock)
}

func CalculateNewBlockTimestamp(prevBlockTimestamp primitives.TimestampNano, now primitives.TimestampNano) primitives.TimestampNano {
	if now > prevBlockTimestamp {
		return now + 1
	}
	return prevBlockTimestamp + 1
}
