package leanhelix

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) leaderProposeNextBlockIfNeeded() error {
	if s.blocksForRounds[s.lastCommittedBlockHeight+1] != nil {
		return nil
	}

	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: 1,
	})
	if err != nil {
		return err
	}

	proposedBlockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{
				ProtocolVersion:       blockstorage.ProtocolVersion,
				BlockHeight:           primitives.BlockHeight(s.lastCommittedBlockHeight + 1),
				NumSignedTransactions: uint32(len(proposedTransactions.SignedTransactions)),
			}).Build(),
			Metadata:           (&protocol.TransactionsBlockMetadataBuilder{}).Build(),
			SignedTransactions: proposedTransactions.SignedTransactions,
			BlockProof:         (&protocol.TransactionsBlockProofBuilder{}).Build(),
		},
		ResultsBlock: &protocol.ResultsBlockContainer{
			Header:              (&protocol.ResultsBlockHeaderBuilder{}).Build(),
			TransactionReceipts: nil,
			ContractStateDiffs:  nil,
			BlockProof:          (&protocol.ResultsBlockProofBuilder{}).Build(),
		},
	}

	s.blocksForRounds[s.lastCommittedBlockHeight+1] = proposedBlockPair
	return nil
}

func (s *service) leaderCollectVotesForBlock(blockPair *protocol.BlockPairContainer) error {
	s.votesForActiveRound = make(chan bool)
	defer func() {
		close(s.votesForActiveRound)
		s.votesForActiveRound = nil
	}()

	_, err := s.gossip.SendLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
		Message: &gossipmessages.LeanHelixPrePrepareMessage{
			SignedHeader: (&consensus.LeanHelixBlockRefBuilder{}).Build(),
			Sender:       (&consensus.LeanHelixSenderSignatureBuilder{}).Build(),
			BlockPair:    blockPair,
		},
	})
	if err != nil {
		return err
	}

	// asking for votes from everybody except ourselves
	for i := 0; i < int(s.config.NetworkSize(0))-1; i++ {
		<-s.votesForActiveRound
	}

	return nil
}

func (s *service) validatorVoteForNewBlockProposal(blockPair *protocol.BlockPairContainer) error {
	blockHeight := blockPair.TransactionsBlock.Header.BlockHeight()
	s.blocksForRounds[blockHeight] = blockPair

	_, err := s.gossip.SendLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
	return err
}

func (s *service) leaderAddVoteFromValidator() {
	// TODO: we assume we only get votes for the active round, in the real world we can't assume this,
	// TODO:  but here since we don't move to the next round unless everybody voted, it's ok
	if s.votesForActiveRound == nil {
		panic("received vote while not collecting votes")
	}
	s.votesForActiveRound <- true
}

func (s *service) commitBlockAndMoveToNextRound() primitives.BlockHeight {
	blockPair, found := s.blocksForRounds[s.lastCommittedBlockHeight+1]
	if !found {
		panic("trying to commit a block that wasn't prepared")
	}

	s.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPair,
	})

	delete(s.blocksForRounds, s.lastCommittedBlockHeight+1)
	return s.lastCommittedBlockHeight + 1
}
