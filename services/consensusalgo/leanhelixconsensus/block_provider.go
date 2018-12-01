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
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"time"
)

type BlockPairWrapper struct {
	blockPair *protocol.BlockPairContainer
}

func (b *BlockPairWrapper) Height() lhprimitives.BlockHeight {
	return lhprimitives.BlockHeight(b.blockPair.TransactionsBlock.Header.BlockHeight())
}

func (b *BlockPairWrapper) BlockHash() lhprimitives.Uint256 {
	// TODO This is surely incorrect, fix to use the right hash
	return lhprimitives.Uint256(b.blockPair.TransactionsBlock.Header.MetadataHash())
}

func ToBlockPairWrapper(blockPair *protocol.BlockPairContainer) *BlockPairWrapper {
	return &BlockPairWrapper{
		blockPair: blockPair,
	}
}

// TODO This should be updated on commit!!!
// Currently put the state of the last committed block here, but it might not be the right place for it.
// See https://tree.taiga.io/project/orbs-network/us/404

// TODO Remove lastCommitedBlock - state must be in lib only
type blockProvider struct {
	logger                              log.BasicLogger
	blockStorage                        services.BlockStorage
	consensusContext                    services.ConsensusContext
	waitTimeForMinimalBlockTransactions time.Duration
	nodePublicKey                       primitives.Ed25519PublicKey
	nodePrivateKey                      primitives.Ed25519PrivateKey
}

func NewBlockProvider(
	logger log.BasicLogger,
	blockStorage services.BlockStorage,
	consensusContext services.ConsensusContext,
	consensusRoundTimeoutInterval time.Duration,
	nodePublicKey primitives.Ed25519PublicKey,
	nodePrivateKey primitives.Ed25519PrivateKey) *blockProvider {

	return &blockProvider{
		logger:                              logger,
		blockStorage:                        blockStorage,
		consensusContext:                    consensusContext,
		waitTimeForMinimalBlockTransactions: consensusRoundTimeoutInterval,
		nodePublicKey:                       nodePublicKey,
		nodePrivateKey:                      nodePrivateKey,
	}

}

func (p *blockProvider) RequestNewBlock(ctx context.Context, prevBlock leanhelix.Block) leanhelix.Block {
	// TODO Is this the right timeout here - probably should be a little smaller
	ctxWithTimeout, cancel := context.WithTimeout(ctx, p.waitTimeForMinimalBlockTransactions)
	defer cancel()

	// TODO: Get prev block - under mutex??

	blockWrapper := prevBlock.(*BlockPairWrapper)

	newBlockHeight := primitives.BlockHeight(prevBlock.Height() + 1)

	p.logger.Info("RequestNewBlock()", log.Stringable("new-block-height", newBlockHeight), log.Stringable("wait-time-for-tx", p.waitTimeForMinimalBlockTransactions))

	// get tx
	txOutput, err := p.consensusContext.RequestNewTransactionsBlock(ctxWithTimeout, &services.RequestNewTransactionsBlockInput{
		BlockHeight:   newBlockHeight,
		PrevBlockHash: digest.CalcTransactionsBlockHash(blockWrapper.blockPair.TransactionsBlock),
	})
	if err != nil {
		return nil
	}

	// get rx
	rxOutput, err := p.consensusContext.RequestNewResultsBlock(ctxWithTimeout, &services.RequestNewResultsBlockInput{
		BlockHeight:       newBlockHeight,
		PrevBlockHash:     digest.CalcResultsBlockHash(blockWrapper.blockPair.ResultsBlock),
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil
	}

	// generate signed block
	// TODO what to do in case of error - similar to handling timeout
	pair, _ := signBlockProposal(txOutput.TransactionsBlock, rxOutput.ResultsBlock, p.nodePrivateKey)
	blockPairWrapper := ToBlockPairWrapper(pair)

	p.logger.Info("RequestNewBlock() returning", log.Int("tx-count", len(txOutput.TransactionsBlock.SignedTransactions)), log.Int("rx-count", len(rxOutput.ResultsBlock.TransactionReceipts)))

	return blockPairWrapper

}

func (p *blockProvider) CalculateBlockHash(block leanhelix.Block) lhprimitives.Uint256 {
	blockPairWrapper := block.(*BlockPairWrapper)
	return deepHash(blockPairWrapper.blockPair.TransactionsBlock, blockPairWrapper.blockPair.ResultsBlock)
}

func deepHash(txBlock *protocol.TransactionsBlockContainer, rxBlock *protocol.ResultsBlockContainer) []byte {
	txHash := digest.CalcTransactionsBlockHash(txBlock)
	rxHash := digest.CalcResultsBlockHash(rxBlock)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}

func (p *blockProvider) ValidateBlock(block leanhelix.Block) bool {
	//TODO Implement me!

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

func (s *service) updateLastCommit(mode handlers.HandleBlockConsensusMode, blockPair *protocol.BlockPairContainer) {
}

func (s *service) validateBlockConsensus(blockPair *protocol.BlockPairContainer, prevCommittedBlockPair *protocol.BlockPairContainer) error {
	// correct block type
	if !blockPair.TransactionsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.TransactionsBlock.BlockProof.Type())
	}
	if !blockPair.ResultsBlock.BlockProof.IsTypeLeanHelix() {
		return errors.Errorf("incorrect block proof type: %v", blockPair.ResultsBlock.BlockProof.Type())
	}

	panic("should not have reached here - not supposed to have generated any Lean Helix blocks!!")

	// TODO Verify with Gad - do this here?
	// prev block hash ptr (if given)
	if prevCommittedBlockPair != nil {
		prevTxHash := digest.CalcTransactionsBlockHash(prevCommittedBlockPair.TransactionsBlock)
		if !blockPair.TransactionsBlock.Header.PrevBlockHashPtr().Equal(prevTxHash) {
			return errors.Errorf("transactions prev block hash does not match prev block: %s", prevTxHash)
		}
		prevRxHash := digest.CalcResultsBlockHash(prevCommittedBlockPair.ResultsBlock)
		if !blockPair.ResultsBlock.Header.PrevBlockHashPtr().Equal(prevRxHash) {
			return errors.Errorf("results prev block hash does not match prev block: %s", prevRxHash)
		}
	}

	// TODO block proof
	//blockProof := blockPair.ResultsBlock.BlockProof.LeanHelix()
	//if !blockProof.Sender().SenderPublicKey().Equal(s.config.ConstantConsensusLeader()) {
	//	return errors.Errorf("block proof not from leader: %s", blockProof.Sender().SenderPublicKey())
	//}
	//signedData := s.dataToSignFrom(blockPair)
	//if !signature.VerifyEd25519(blockProof.Sender().SenderPublicKey(), signedData, blockProof.Sender().Signature()) {
	//	return errors.Errorf("block proof signature is invalid: %s", blockProof.Sender().Signature())
	//}

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

	// TODO Fill BlockProof here once implemented in LH lib
	// generate tx block proof
	blockPair.TransactionsBlock.BlockProof = (&protocol.TransactionsBlockProofBuilder{
		Type:      protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: &consensus.LeanHelixBlockProofBuilder{},
	}).Build()

	// generate rx block proof
	blockPair.ResultsBlock.BlockProof = (&protocol.ResultsBlockProofBuilder{
		Type:      protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX,
		LeanHelix: &consensus.LeanHelixBlockProofBuilder{},
	}).Build()
	return blockPair, nil
}

func dataToSignFrom(blockPair *protocol.BlockPairContainer) []byte {
	return deepHash(blockPair.TransactionsBlock, blockPair.ResultsBlock)
}
