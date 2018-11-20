package leanhelixconsensus

import (
	"context"
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
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

func NewBlockPairWrapper(blockPair *protocol.BlockPairContainer) *BlockPairWrapper {
	return &BlockPairWrapper{
		blockPair: blockPair,
	}
}

// TODO This should be updated on commit!!!
// Currently put the state of the last committed block here, but it might not be the right place for it.
// See https://tree.taiga.io/project/orbs-network/us/404
type blockProvider struct {
	consensusContext              services.ConsensusContext
	consensusRoundTimeoutInterval time.Duration
	nodePublicKey                 primitives.Ed25519PublicKey
	nodePrivateKey                primitives.Ed25519PrivateKey
	lastCommittedBlock            *protocol.BlockPairContainer
	lastCommittedBlockHeight      primitives.BlockHeight
}

func NewBlockProvider(consensusRoundTimeoutInterval time.Duration, nodePublicKey primitives.Ed25519PublicKey, nodePrivateKey primitives.Ed25519PrivateKey) *blockProvider {

	return &blockProvider{
		consensusRoundTimeoutInterval: consensusRoundTimeoutInterval,
		nodePublicKey:                 nodePublicKey,
		nodePrivateKey:                nodePrivateKey,
	}

}

func (p *blockProvider) RequestNewBlock(ctx context.Context, blockHeight lhprimitives.BlockHeight) leanhelix.Block {
	// TODO Is this the right timeout here - probably should be a little smaller
	ctxWithTimeout, cancel := context.WithTimeout(ctx, p.consensusRoundTimeoutInterval)
	defer cancel()

	//lastCommittedBlockHeight, lastCommittedBlock := s.getLastCommittedBlock()
	//logger.Info("generating new proposed block", log.BlockHeight(lastCommittedBlockHeight+1))

	// get tx
	txOutput, err := p.consensusContext.RequestNewTransactionsBlock(ctxWithTimeout, &services.RequestNewTransactionsBlockInput{
		BlockHeight:   p.lastCommittedBlockHeight + 1,
		PrevBlockHash: digest.CalcTransactionsBlockHash(p.lastCommittedBlock.TransactionsBlock),
	})
	if err != nil {
		return nil
	}

	// get rx
	rxOutput, err := p.consensusContext.RequestNewResultsBlock(ctxWithTimeout, &services.RequestNewResultsBlockInput{
		BlockHeight:       p.lastCommittedBlockHeight + 1,
		PrevBlockHash:     digest.CalcResultsBlockHash(p.lastCommittedBlock.ResultsBlock),
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil
	}

	// generate signed block
	// TODO what to do in case of error - similar to handling timeout
	pair, _ := p.signBlockProposal(txOutput.TransactionsBlock, rxOutput.ResultsBlock)
	blockPairWrapper := NewBlockPairWrapper(&protocol.BlockPairContainer{
		TransactionsBlock: pair.TransactionsBlock,
		ResultsBlock:      pair.ResultsBlock,
	})

	return blockPairWrapper

}

func (p *blockProvider) CalculateBlockHash(block leanhelix.Block) lhprimitives.Uint256 {
	blockPairWrapper := block.(*BlockPairWrapper)
	return p.hash(blockPairWrapper.blockPair.TransactionsBlock, blockPairWrapper.blockPair.ResultsBlock)
}

func (p *blockProvider) hash(txBlock *protocol.TransactionsBlockContainer, rxBlock *protocol.ResultsBlockContainer) []byte {
	txHash := digest.CalcTransactionsBlockHash(txBlock)
	rxHash := digest.CalcResultsBlockHash(rxBlock)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}

func (p *blockProvider) ValidateBlock(block leanhelix.Block) bool {
	panic("implement me")
}

// This calls ValidateBlockConsensus
func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {

	blockType := input.BlockType
	mode := input.Mode
	blockPair := input.BlockPair
	prevCommittedBlockPair := input.PrevCommittedBlockPair
	if blockType != protocol.BLOCK_TYPE_BLOCK_PAIR {
		return nil, errors.Errorf("handler received unsupported block type %s", blockType)
	}

	// validate the block consensus
	if mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE || mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY {
		err := s.validateBlockConsensus(blockPair, prevCommittedBlockPair)
		if err != nil {
			return nil, err
		}
	}

	// update lastCommitted to reflect this if newer
	// TODO: Gad - how to update internal block height and continue from there
	if mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE || mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY {
		//lastCommittedBlockHeight, lastCommittedBlock := s.getLastCommittedBlock()
		//
		//
		//if blockPair.TransactionsBlock.Header.BlockHeight() > lastCommittedBlockHeight {
		//
		//	// TODO Set last committed block here?
		//	//err := s.setLastCommittedBlock(blockPair, lastCommittedBlock)
		//	//if err != nil {
		//	//	return err
		//	//}
		//	// don't forget to update internal vars too since they may be used later on in the function
		//	lastCommittedBlock = blockPair
		//	lastCommittedBlockHeight = lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
		//}
	}

	return nil, nil
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

func (s *service) getLastCommittedBlock() (primitives.BlockHeight, *protocol.BlockPairContainer) {
	s.lastCommittedBlock.RLock()
	defer s.lastCommittedBlock.RUnlock()

	if s.lastCommittedBlock.block == nil {
		return 0, nil
	}
	return s.lastCommittedBlock.block.TransactionsBlock.Header.BlockHeight(), s.lastCommittedBlock.block
}

func (p *blockProvider) signBlockProposal(transactionsBlock *protocol.TransactionsBlockContainer, resultsBlock *protocol.ResultsBlockContainer) (*protocol.BlockPairContainer, error) {
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock:      resultsBlock,
	}

	// prepare signature over the block headers
	blockPairDataToSign := p.dataToSignFrom(blockPair)
	sig, err := signature.SignEd25519(p.nodePrivateKey, blockPairDataToSign)
	if err != nil {
		return nil, err
	}

	// generate tx block proof
	blockPair.TransactionsBlock.BlockProof = (&protocol.TransactionsBlockProofBuilder{
		Type:               protocol.TRANSACTIONS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{},
	}).Build()

	// generate rx block proof
	blockPair.ResultsBlock.BlockProof = (&protocol.ResultsBlockProofBuilder{
		Type: protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{
			Sender: &consensus.BenchmarkConsensusSenderSignatureBuilder{
				SenderPublicKey: p.nodePublicKey,
				Signature:       sig,
			},
		},
	}).Build()
	return blockPair, nil
}

func (p *blockProvider) dataToSignFrom(blockPair *protocol.BlockPairContainer) []byte {
	return p.hash(blockPair.TransactionsBlock, blockPair.ResultsBlock)
}
