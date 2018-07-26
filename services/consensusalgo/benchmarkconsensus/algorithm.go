package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
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
				time.Sleep(1 * time.Second) // TODO: replace with a configuration
			}
		}
	}
}

func (s *service) consensusRoundTick() (err error) {
	s.reporting.Infof("Entered consensus round, last committed block height is %d", s.lastCommittedBlockHeight())
	if s.activeBlock == nil {
		s.activeBlock, err = s.generateNewProposedBlock()
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

func (s *service) generateNewProposedBlock() (*protocol.BlockPairContainer, error) {
	_, err := s.consensusContext.RequestNewTransactionsBlock(&services.RequestNewTransactionsBlockInput{})
	if err != nil {
		return nil, err
	}
	return nil, nil
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
	txHash := crypto.CalcTransactionsBlockHash(blockPair)
	rxHash := crypto.CalcResultsBlockHash(blockPair)
	xorHash := logic.CalcXor(txHash, rxHash)
	if !signature.VerifyEd25519(blockProof.Sender().SenderPublicKey(), xorHash, blockProof.Sender().Signature()) {
		return errors.Errorf("block proof signature is invalid: %s", blockProof.Sender().Signature())
	}
	return nil
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
