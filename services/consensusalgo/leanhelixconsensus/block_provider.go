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

func (s *service) RequestNewBlock(parentCtx context.Context, blockHeight lhprimitives.BlockHeight) leanhelix.Block {

	// TODO Is this the right timeout here - probably should be a little smaller
	ctxWithTimeout, cancel := context.WithTimeout(parentCtx, s.config.LeanHelixConsensusRoundTimeoutInterval())
	defer cancel()

	lastCommittedBlockHeight, lastCommittedBlock := s.getLastCommittedBlock()
	s.logger.Info("generating new proposed block", log.BlockHeight(lastCommittedBlockHeight+1))

	// get tx
	txOutput, err := s.consensusContext.RequestNewTransactionsBlock(ctxWithTimeout, &services.RequestNewTransactionsBlockInput{
		BlockHeight:   lastCommittedBlockHeight + 1,
		PrevBlockHash: digest.CalcTransactionsBlockHash(lastCommittedBlock.TransactionsBlock),
	})
	if err != nil {
		return nil
	}

	// get rx
	rxOutput, err := s.consensusContext.RequestNewResultsBlock(ctxWithTimeout, &services.RequestNewResultsBlockInput{
		BlockHeight:       lastCommittedBlockHeight + 1,
		PrevBlockHash:     digest.CalcResultsBlockHash(lastCommittedBlock.ResultsBlock),
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil
	}

	// generate signed block
	// TODO what to do in case of error - similar to handling timeout
	pair, _ := s.signBlockProposal(txOutput.TransactionsBlock, rxOutput.ResultsBlock)
	blockPairWrapper := NewBlockPairWrapper(&protocol.BlockPairContainer{
		TransactionsBlock: pair.TransactionsBlock,
		ResultsBlock:      pair.ResultsBlock,
	})

	return blockPairWrapper

}

func (s *service) signBlockProposal(transactionsBlock *protocol.TransactionsBlockContainer, resultsBlock *protocol.ResultsBlockContainer) (*protocol.BlockPairContainer, error) {
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock:      resultsBlock,
	}

	// prepare signature over the block headers
	blockPairDataToSign := s.dataToSignFrom(blockPair)
	sig, err := signature.SignEd25519(s.config.NodePrivateKey(), blockPairDataToSign)
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
				SenderPublicKey: s.config.NodePublicKey(),
				Signature:       sig,
			},
		},
	}).Build()
	return blockPair, nil
}

func (s *service) hash(txBlock *protocol.TransactionsBlockContainer, rxBlock *protocol.ResultsBlockContainer) []byte {
	txHash := digest.CalcTransactionsBlockHash(txBlock)
	rxHash := digest.CalcResultsBlockHash(rxBlock)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}

func (s *service) dataToSignFrom(blockPair *protocol.BlockPairContainer) []byte {
	return s.hash(blockPair.TransactionsBlock, blockPair.ResultsBlock)
}

func (s *service) CalculateBlockHash(block leanhelix.Block) lhprimitives.Uint256 {
	blockPairWrapper := block.(*BlockPairWrapper)
	return s.hash(blockPairWrapper.blockPair.TransactionsBlock, blockPairWrapper.blockPair.ResultsBlock)
}
