package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"time"
)

func (s *service) consensusRoundRunLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			s.reporting.Infof("Consensus round run loop terminating with context")
			return
		default:
			err := s.consensusRoundTick()
			if err != nil {
				s.reporting.Error(err)
				time.Sleep(time.Duration(s.config.BenchmarkConsensusRoundRetryIntervalMillisec()) * time.Millisecond)
			}
		}
	}
}

func (s *service) consensusRoundTick() (err error) {
	s.reporting.Infof("Entered consensus round, last committed block height is %d", s.lastCommittedBlockHeight())
	if s.activeBlock == nil {
		s.activeBlock, err = s.leaderGenerateNewProposedBlock()
		if err != nil {
			return err
		}
	}
	if s.activeBlock != nil {
		_, err = s.gossip.BroadcastBenchmarkConsensusCommit(&gossiptopics.BenchmarkConsensusCommitInput{
			Message: &gossipmessages.BenchmarkConsensusCommitMessage{
				BlockPair: s.activeBlock,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) lastCommittedBlockHeight() primitives.BlockHeight {
	if s.lastCommittedBlock == nil {
		return 0
	}
	return s.lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
}

func (s *service) leaderGenerateNewProposedBlock() (*protocol.BlockPairContainer, error) {
	// get tx
	txOutput, err := s.consensusContext.RequestNewTransactionsBlock(&services.RequestNewTransactionsBlockInput{
		BlockHeight: s.lastCommittedBlockHeight() + 1,
	})
	if err != nil {
		return nil, err
	}

	// get rx
	rxOutput, err := s.consensusContext.RequestNewResultsBlock(&services.RequestNewResultsBlockInput{
		BlockHeight: s.lastCommittedBlockHeight() + 1,
	})
	if err != nil {
		return nil, err
	}

	// generate block
	if txOutput == nil || txOutput.TransactionsBlock == nil || rxOutput == nil || rxOutput.ResultsBlock == nil {
		panic("invalid responses: missing fields")
		// TODO: should we have these panics? because this is internal code
	}
	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: txOutput.TransactionsBlock,
		ResultsBlock:      rxOutput.ResultsBlock,
	}

	// generate block proof
	signedData := s.signedDataForBlockProof(blockPair)
	signature := signature.SignEd25519(nil, signedData)
	blockPair.ResultsBlock.BlockProof = (&protocol.ResultsBlockProofBuilder{
		Type: protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS,
		BenchmarkConsensus: &consensus.BenchmarkConsensusBlockProofBuilder{
			Sender: &consensus.BenchmarkConsensusSenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
				Signature:       signature,
			},
		},
	}).Build()

	return blockPair, nil
}

func (s *service) nonLeaderHandleCommit(blockPair *protocol.BlockPairContainer) {
	err := s.nonLeaderValidateBlock(blockPair)
	if err != nil {
		s.reporting.Error(err) // TODO: wrap with added context
		return
	}
	err = s.nonLeaderCommitAndReply(blockPair)
	if err != nil {
		s.reporting.Error(err) // TODO: wrap with added context
		return
	}
}

func (s *service) nonLeaderValidateBlock(blockPair *protocol.BlockPairContainer) error {
	// nils
	if blockPair.TransactionsBlock == nil ||
		blockPair.ResultsBlock == nil ||
		blockPair.TransactionsBlock.Header == nil ||
		blockPair.ResultsBlock.Header == nil ||
		blockPair.ResultsBlock.BlockProof == nil {
		panic("invalid block: missing fields")
	}

	// block height
	blockHeight := blockPair.TransactionsBlock.Header.BlockHeight()
	if blockHeight > s.lastCommittedBlockHeight()+1 {
		return errors.Errorf("invalid block: future block height %d", blockHeight)
	}

	// correct block type
	if !blockPair.ResultsBlock.BlockProof.IsTypeBenchmarkConsensus() {
		return errors.Errorf("incorrect block proof type: %s", blockPair.ResultsBlock.BlockProof.Type())
	}

	// prev block hash ptr
	if s.lastCommittedBlock != nil && blockHeight == s.lastCommittedBlockHeight()+1 {
		prevTxHash := crypto.CalcTransactionsBlockHash(s.lastCommittedBlock)
		if !blockPair.TransactionsBlock.Header.PrevBlockHashPtr().Equal(prevTxHash) {
			return errors.Errorf("transactions prev block hash does not match prev block: %s", prevTxHash)
		}
		prevRxHash := crypto.CalcResultsBlockHash(s.lastCommittedBlock)
		if !blockPair.ResultsBlock.Header.PrevBlockHashPtr().Equal(prevRxHash) {
			return errors.Errorf("results prev block hash does not match prev block: %s", prevRxHash)
		}
	}

	// block proof
	blockProof := blockPair.ResultsBlock.BlockProof.BenchmarkConsensus()
	if !blockProof.Sender().SenderPublicKey().Equal(s.config.ConstantConsensusLeader()) {
		return errors.Errorf("block proof not from leader: %s", blockProof.Sender().SenderPublicKey())
	}
	signedData := s.signedDataForBlockProof(blockPair)
	if !signature.VerifyEd25519(blockProof.Sender().SenderPublicKey(), signedData, blockProof.Sender().Signature()) {
		return errors.Errorf("block proof signature is invalid: %s", blockProof.Sender().Signature())
	}
	return nil
}

func (s *service) signedDataForBlockProof(blockPair *protocol.BlockPairContainer) []byte {
	txHash := crypto.CalcTransactionsBlockHash(blockPair)
	rxHash := crypto.CalcResultsBlockHash(blockPair)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}

func (s *service) nonLeaderCommitAndReply(blockPair *protocol.BlockPairContainer) error {
	_, err := s.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPair,
	})
	if err != nil {
		return err
	}
	if blockPair.TransactionsBlock.Header.BlockHeight() == s.lastCommittedBlockHeight()+1 {
		s.lastCommittedBlock = blockPair
	}
	_, err = s.gossip.SendBenchmarkConsensusCommitted(&gossiptopics.BenchmarkConsensusCommittedInput{
		RecipientPublicKey: blockPair.ResultsBlock.BlockProof.BenchmarkConsensus().Sender().SenderPublicKey(),
		Message: &gossipmessages.BenchmarkConsensusCommittedMessage{
			Status: (&gossipmessages.BenchmarkConsensusStatusBuilder{
				LastCommittedBlockHeight: s.lastCommittedBlockHeight(),
			}).Build(),
			Sender: nil,
		},
	})
	return err
}
