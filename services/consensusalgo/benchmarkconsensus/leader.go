package benchmarkconsensus

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"time"
)

func (s *service) leaderConsensusRoundRunLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			// TODO: in production we need to restart our long running goroutine (decide on supervision mechanism)
			s.logger.Error("panic in BenchmarkConsensus.leaderConsensusRoundRunLoop long running goroutine", log.String("panic", fmt.Sprintf("%v", r)))
		}
	}()

	s.lastCommittedBlock = s.leaderGenerateGenesisBlock()
	for {
		err := s.leaderConsensusRoundTick(ctx)
		if err != nil {
			s.logger.Error("consensus round tick failed", log.Error(err))
			s.metrics.failedConsensusTicksRate.Measure(1)
		}
		select {
		case <-ctx.Done():
			s.logger.Info("consensus round run loop terminating with context")
			// FIXME remove the channel close once we start passing context everywhere
			// TODO (talkol) - it's a pattern we need to decide on: many short lived writers, one long lived reader
			// closing the channel when long lived reader terminates will cause the writers to panic - a smell
			// the better fix is to send ctx to all writers and when they block write, select on the ctx.Done as well
			// we can only implement this once ctx can be sent to the writers
			close(s.successfullyVotedBlocks)
			return
		case s.lastSuccessfullyVotedBlock = <-s.successfullyVotedBlocks:
			s.logger.Info("consensus round waking up after successfully voted block", log.BlockHeight(s.lastSuccessfullyVotedBlock))
			continue
		case <-time.After(s.config.BenchmarkConsensusRetryInterval()):
			s.logger.Info("consensus round waking up after retry timeout")
			s.metrics.timedOutConsensusTicksRate.Measure(1)
			continue
		}
	}
}

func (s *service) leaderConsensusRoundTick(ctx context.Context) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	start := time.Now()
	defer s.metrics.consensusRoundTickTime.RecordSince(start)

	// check if we need to move to next block
	if s.lastSuccessfullyVotedBlock == s.lastCommittedBlockHeightUnderMutex() {
		proposedBlock, err := s.leaderGenerateNewProposedBlockUnderMutex(ctx)
		if err != nil {
			return err
		}
		err = s.saveToBlockStorage(ctx, proposedBlock)
		if err != nil {
			return err
		}

		s.lastCommittedBlock = proposedBlock
		s.lastCommittedBlockVoters = make(map[string]bool)
		s.lastCommittedBlockVotersReachedQuorum = false
	}

	// broadcast the commit via gossip for last committed block
	err = s.leaderBroadcastCommittedBlock(ctx, s.lastCommittedBlock)
	if err != nil {
		return err
	}

	if s.config.NetworkSize(0) == 1 {
		s.successfullyVotedBlocks <- s.lastCommittedBlockHeightUnderMutex()
	}

	return nil
}

// used for the first commit a leader does which is nop (genesis block) just to see where everybody's at
func (s *service) leaderGenerateGenesisBlock() *protocol.BlockPairContainer {
	transactionsBlock := &protocol.TransactionsBlockContainer{
		Header:             (&protocol.TransactionsBlockHeaderBuilder{BlockHeight: 0}).Build(),
		Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
		SignedTransactions: []*protocol.SignedTransaction{},
		BlockProof:         nil, // will be generated in a minute when signed
	}
	resultsBlock := &protocol.ResultsBlockContainer{
		Header:              (&protocol.ResultsBlockHeaderBuilder{BlockHeight: 0}).Build(),
		TransactionReceipts: []*protocol.TransactionReceipt{},
		ContractStateDiffs:  []*protocol.ContractStateDiff{},
		BlockProof:          nil, // will be generated in a minute when signed
	}
	blockPair, err := s.leaderSignBlockProposal(transactionsBlock, resultsBlock)
	if err != nil {
		s.logger.Error("leader failed to sign genesis block", log.Error(err))
		return nil
	}
	return blockPair
}

func (s *service) leaderGenerateNewProposedBlockUnderMutex(ctx context.Context) (*protocol.BlockPairContainer, error) {
	s.logger.Info("generating new proposed block", log.BlockHeight(s.lastCommittedBlockHeightUnderMutex()+1))

	// get tx
	txOutput, err := s.consensusContext.RequestNewTransactionsBlock(ctx, &services.RequestNewTransactionsBlockInput{
		BlockHeight:   s.lastCommittedBlockHeightUnderMutex() + 1,
		PrevBlockHash: digest.CalcTransactionsBlockHash(s.lastCommittedBlock.TransactionsBlock),
	})
	if err != nil {
		return nil, err
	}

	// get rx
	rxOutput, err := s.consensusContext.RequestNewResultsBlock(ctx, &services.RequestNewResultsBlockInput{
		BlockHeight:       s.lastCommittedBlockHeightUnderMutex() + 1,
		PrevBlockHash:     digest.CalcResultsBlockHash(s.lastCommittedBlock.ResultsBlock),
		TransactionsBlock: txOutput.TransactionsBlock,
	})
	if err != nil {
		return nil, err
	}

	// generate signed block
	return s.leaderSignBlockProposal(txOutput.TransactionsBlock, rxOutput.ResultsBlock)
}

func (s *service) leaderSignBlockProposal(transactionsBlock *protocol.TransactionsBlockContainer, resultsBlock *protocol.ResultsBlockContainer) (*protocol.BlockPairContainer, error) {
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

func (s *service) leaderBroadcastCommittedBlock(ctx context.Context, blockPair *protocol.BlockPairContainer) error {
	s.logger.Info("broadcasting commit block", log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))

	// the block pair fields we have may be partial (for example due to being read from persistence storage on init) so don't broadcast it in this case
	if blockPair == nil || blockPair.TransactionsBlock.BlockProof == nil || blockPair.ResultsBlock.BlockProof == nil {
		return errors.Errorf("attempting to broadcast commit of a partial block that is missing fields like block proofs: %v", blockPair.String())
	}

	_, err := s.gossip.BroadcastBenchmarkConsensusCommit(ctx, &gossiptopics.BenchmarkConsensusCommitInput{
		Message: &gossipmessages.BenchmarkConsensusCommitMessage{
			BlockPair: blockPair,
		},
	})

	return err
}

func (s *service) leaderHandleCommittedVote(sender *gossipmessages.SenderSignature, status *gossipmessages.BenchmarkConsensusStatus) {
	defer func() {
		// FIXME remove the recover once we start passing context everywhere
		// TODO (talkol) - it's a pattern we need to decide on: many short lived writers, one long lived reader
		// closing the channel when long lived reader terminates will cause the writers to panic - a smell
		// the better fix is to send ctx to all writers and when they block write, select on the ctx.Done as well
		// we can only implement this once ctx can be sent to the writers
		if r := recover(); r != nil {
			fields := []*log.Field{}
			if err, ok := r.(error); ok {
				fields = append(fields, log.Error(err))
			}

			s.logger.Info("recovering from failure to collect vote, possibly because consensus was shut down", fields...)
		}
	}()

	successfullyVotedBlock := blockHeightNone
	defer func() {
		// this needs to happen after s.mutex.Unlock() to avoid deadlock
		if successfullyVotedBlock != blockHeightNone {
			s.successfullyVotedBlocks <- successfullyVotedBlock
		}
	}()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// validate the vote
	err := s.leaderValidateVoteUnderMutex(sender, status)
	if err != nil {
		s.logger.Error("leader failed to validate vote", log.Error(err))
		return
	}

	// add the vote
	s.lastCommittedBlockVoters[sender.SenderPublicKey().KeyForMap()] = true

	// count if we have enough votes to move forward
	existingVotes := len(s.lastCommittedBlockVoters) + 1
	s.logger.Info("valid vote arrived", log.BlockHeight(status.LastCommittedBlockHeight()), log.Int("existing-votes", existingVotes), log.Int("required-votes", s.requiredQuorumSize()))
	if existingVotes >= s.requiredQuorumSize() && !s.lastCommittedBlockVotersReachedQuorum {
		s.lastCommittedBlockVotersReachedQuorum = true
		successfullyVotedBlock = s.lastCommittedBlockHeightUnderMutex()
	}
}

func (s *service) leaderValidateVoteUnderMutex(sender *gossipmessages.SenderSignature, status *gossipmessages.BenchmarkConsensusStatus) error {
	// block height
	blockHeight := status.LastCommittedBlockHeight()
	if blockHeight != s.lastCommittedBlockHeightUnderMutex() {
		return errors.Errorf("committed message with wrong block height %d, expecting %d", blockHeight, s.lastCommittedBlockHeightUnderMutex())
	}

	// approved signer
	if _, found := s.config.FederationNodes(0)[sender.SenderPublicKey().KeyForMap()]; !found {
		return errors.Errorf("signer with public key %s is not a valid federation member", sender.SenderPublicKey())
	}

	// signature
	signedData := hash.CalcSha256(status.Raw())
	if !signature.VerifyEd25519(sender.SenderPublicKey(), signedData, sender.Signature()) {
		return errors.Errorf("sender signature is invalid: %s, signed data: %s", sender.Signature(), signedData)
	}

	return nil
}
