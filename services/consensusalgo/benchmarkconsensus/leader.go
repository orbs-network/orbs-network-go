package benchmarkconsensus

import (
	"context"
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
	s.lastCommittedBlock = s.leaderGenerateGenesisBlock()
	for {
		err := s.leaderConsensusRoundTick()
		if err != nil {
			s.reporting.Error(err.Error())
		}
		select {
		case <-ctx.Done():
			s.reporting.Info("consensus round run loop terminating with context")
			return
		case s.lastSuccessfullyVotedBlock = <-s.successfullyVotedBlocks:
			s.reporting.Info("consensus round waking up after successfully voted block", log.BlockHeight(s.lastSuccessfullyVotedBlock))
			continue
		case <-time.After(s.config.BenchmarkConsensusRetryInterval()):
			s.reporting.Info("consensus round waking up after retry timeout")
			continue
		}
	}
}

func (s *service) leaderConsensusRoundTick() (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// check if we need to move to next block
	if s.lastSuccessfullyVotedBlock == s.lastCommittedBlockHeightUnderMutex() {
		proposedBlock, err := s.leaderGenerateNewProposedBlockUnderMutex()
		if err != nil {
			return err
		}
		err = s.saveToBlockStorage(proposedBlock)
		if err != nil {
			return err
		}

		s.lastCommittedBlock = proposedBlock
		s.lastCommittedBlockVoters = make(map[string]bool)
	}

	// broadcast the commit via gossip for last committed block
	err = s.leaderBroadcastCommittedBlock(s.lastCommittedBlock)
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
		s.reporting.Error("leader failed to sign genesis block", log.Error(err))
		return nil
	}
	return blockPair
}

func (s *service) leaderGenerateNewProposedBlockUnderMutex() (*protocol.BlockPairContainer, error) {
	s.reporting.Info("generating new proposed block", log.BlockHeight(s.lastCommittedBlockHeightUnderMutex()+1))

	// get tx
	txOutput, err := s.consensusContext.RequestNewTransactionsBlock(&services.RequestNewTransactionsBlockInput{
		BlockHeight:   s.lastCommittedBlockHeightUnderMutex() + 1,
		PrevBlockHash: digest.CalcTransactionsBlockHash(s.lastCommittedBlock.TransactionsBlock),
	})
	if err != nil {
		return nil, err
	}

	// get rx
	rxOutput, err := s.consensusContext.RequestNewResultsBlock(&services.RequestNewResultsBlockInput{
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

func (s *service) leaderBroadcastCommittedBlock(blockPair *protocol.BlockPairContainer) error {
	s.reporting.Info("broadcasting commit block", log.BlockHeight(blockPair.TransactionsBlock.Header.BlockHeight()))

	// the block pair fields we have may be partial (for example due to being read from persistence storage on init) so don't broadcast it in this case
	if blockPair == nil || blockPair.TransactionsBlock.BlockProof == nil || blockPair.ResultsBlock.BlockProof == nil {
		return errors.Errorf("attempting to broadcast commit of a partial block that is missing fields like block proofs: %v", blockPair.String())
	}

	_, err := s.gossip.BroadcastBenchmarkConsensusCommit(&gossiptopics.BenchmarkConsensusCommitInput{
		Message: &gossipmessages.BenchmarkConsensusCommitMessage{
			BlockPair: blockPair,
		},
	})

	return err
}

func (s *service) leaderHandleCommittedVote(sender *gossipmessages.SenderSignature, status *gossipmessages.BenchmarkConsensusStatus) {
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
		s.reporting.Error("leader failed to validate vote", log.Error(err))
		return
	}

	// add the vote
	s.lastCommittedBlockVoters[sender.SenderPublicKey().KeyForMap()] = true

	// count if we have enough votes to move forward
	existingVotes := len(s.lastCommittedBlockVoters) + 1
	s.reporting.Info("valid vote arrived", log.BlockHeight(status.LastCommittedBlockHeight()), log.Int("existing-votes", existingVotes), log.Int("required-votes", s.requiredQuorumSize()))
	if existingVotes >= s.requiredQuorumSize() {
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
