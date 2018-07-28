package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"math"
	"time"
)

func (s *service) consensusRoundRunLoop(ctx context.Context) {
	for {
		err := s.leaderConsensusRoundTick()
		if err != nil {
			s.reporting.Error(err)
		}
		select {
		case <-ctx.Done():
			s.reporting.Info("Consensus round run loop terminating with context")
			return
		case s.lastSuccessfullyVotedBlock = <-s.successfullyVotedBlocks:
			s.reporting.Infof("Consensus round waking up after successfully voted block %d", s.lastSuccessfullyVotedBlock)
			continue
		case <-time.After(time.Duration(s.config.BenchmarkConsensusRoundRetryIntervalMillisec()) * time.Millisecond):
			s.reporting.Info("Consensus round waking up after retry timeout")
			continue
		}
	}
}

func (s *service) leaderConsensusRoundTick() (err error) {
	s.leaderMutex.Lock()
	defer s.leaderMutex.Unlock()

	// check if we need to move to next block
	if s.lastSuccessfullyVotedBlock == s.lastCommittedBlockHeight() {
		proposedBlock, err := s.leaderGenerateNewProposedBlockUnsafe()
		if err != nil {
			return err
		}
		s.lastCommittedBlock = proposedBlock
		s.lastCommittedBlockVoters = make(map[string]bool)
		// TODO: commit this block locally
	}

	// broadcast the commit via gossip for last committed block
	if s.lastCommittedBlock != nil {
		_, err = s.gossip.BroadcastBenchmarkConsensusCommit(&gossiptopics.BenchmarkConsensusCommitInput{
			Message: &gossipmessages.BenchmarkConsensusCommitMessage{
				BlockPair: s.lastCommittedBlock,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *service) leaderGenerateNewProposedBlockUnsafe() (*protocol.BlockPairContainer, error) {
	s.reporting.Infof("Generating new proposed block for height %d", s.lastCommittedBlockHeight()+1)

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
	sig := signature.SignEd25519(nil, signedData)
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

func (s *service) leaderHandleCommittedVote(sender *gossipmessages.SenderSignature, status *gossipmessages.BenchmarkConsensusStatus) {
	successfullyVotedBlock := primitives.BlockHeight(0)
	defer func() {
		// this needs to happen after s.leaderMutex.Unlock() to avoid deadlock
		if successfullyVotedBlock > 0 {
			s.successfullyVotedBlocks <- successfullyVotedBlock
		}
	}()

	s.leaderMutex.Lock()
	defer s.leaderMutex.Unlock()

	// validate the vote
	err := s.leaderValidateVoteUnsafe(sender, status)
	if err != nil {
		s.reporting.Error(err) // TODO: wrap with added context
		return
	}

	// add the vote
	s.lastCommittedBlockVoters[sender.SenderPublicKey().KeyForMap()] = true

	// count if we have enough votes to move forward
	existingVotes := len(s.lastCommittedBlockVoters) + 1
	neededVotes := int(math.Ceil(float64(s.config.NetworkSize(0)) * 2 / 3))
	s.reporting.Infof("Vote arrived, now have %d votes out of %d needed", existingVotes, neededVotes)
	if existingVotes >= neededVotes {
		successfullyVotedBlock = s.lastCommittedBlockHeight()
	}
}

func (s *service) leaderValidateVoteUnsafe(sender *gossipmessages.SenderSignature, status *gossipmessages.BenchmarkConsensusStatus) error {
	// block height
	blockHeight := status.LastCommittedBlockHeight()
	if blockHeight != s.lastCommittedBlockHeight() {
		return errors.Errorf("committed message with wrong block height %d, expecting %d", blockHeight, s.lastCommittedBlockHeight())
	}

	// signature
	// TODO: check if an approved sender
	signedData := hash.CalcSha256(status.Raw())
	if !signature.VerifyEd25519(sender.SenderPublicKey(), signedData, sender.Signature()) {
		return errors.Errorf("sender signature is invalid: %s", sender.Signature())
	}

	return nil
}
