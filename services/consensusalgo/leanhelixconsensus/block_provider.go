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
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func (s *service) HandleBlockConsensus(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
	panic("implement validate block consensus (call the lib)")
}

func (s *service) HandleLeanHelixMessage(ctx context.Context, input *gossiptopics.LeanHelixInput) (*gossiptopics.EmptyOutput, error) {
	panic("implement me")
}

func (s *service) getLastCommittedBlock() (primitives.BlockHeight, *protocol.BlockPairContainer) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.lastCommittedBlockUnderMutex == nil {
		return 0, nil
	}
	return s.lastCommittedBlockUnderMutex.TransactionsBlock.Header.BlockHeight(), s.lastCommittedBlockUnderMutex
}

func (s *service) RequestNewBlock(ctx context.Context, blockHeight lhprimitives.BlockHeight) (leanhelix.Block, error) {

	_lastCommittedBlockHeight, _lastCommittedBlock := s.getLastCommittedBlock()
	s.logger.Info("generating new proposed block", log.BlockHeight(_lastCommittedBlockHeight+1))

	// get tx
	txOutput, err := s.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
		BlockHeight:   _lastCommittedBlockHeight + 1,
		PrevBlockHash: digest.CalcTransactionsBlockHash(_lastCommittedBlock.TransactionsBlock),
	})
	if err != nil {
		return nil, err
	}

	// get rx
	rxOutput, err := s.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
		BlockHeight:       _lastCommittedBlockHeight + 1,
		PrevBlockHash:     digest.CalcResultsBlockHash(_lastCommittedBlock.ResultsBlock),
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil, err
	}

	// generate signed block
	pair, err := s.SignBlockProposal(txOutput.TransactionsBlock, rxOutput.ResultsBlock)
	blockPairWrapper := NewBlockPairWrapper(&protocol.BlockPairContainer{
		TransactionsBlock: pair.TransactionsBlock,
		ResultsBlock:      pair.ResultsBlock,
	})

	return blockPairWrapper, nil

}

func (s *service) SignBlockProposal(transactionsBlock *protocol.TransactionsBlockContainer, resultsBlock *protocol.ResultsBlockContainer) (*protocol.BlockPairContainer, error) {
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: transactionsBlock,
		ResultsBlock:      resultsBlock,
	}

	// prepare signature over the block headers
	signedData := s.signedDataForBlockProof(blockPair)
	sig, err := signature.SignEd25519(s.config.NodePrivateKey(), signedData)
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

func (s *service) signedDataForBlockProof(blockPair *protocol.BlockPairContainer) []byte {
	return s.hash(blockPair.TransactionsBlock, blockPair.ResultsBlock)
}

func (s *service) signedDataForBlockProofWrapper(blockPairWrapper *BlockPairWrapper) []byte {
	return s.hash(blockPairWrapper.blockPair.TransactionsBlock, blockPairWrapper.blockPair.ResultsBlock)
}

func (s *service) CalculateBlockHash(block leanhelix.Block) lhprimitives.Uint256 {

	return s.signedDataForBlockProofWrapper(block.(*BlockPairWrapper))
}
