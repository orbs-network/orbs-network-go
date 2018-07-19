package leanhelix

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func (s *service) leaderProposeNextBlock() (*protocol.BlockPairContainer, error) {
	proposedTransactions, err := s.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: 1,
	})
	if err != nil {
		return nil, err
	}

	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{
				ProtocolVersion: blockstorage.ProtocolVersion,
				BlockHeight:     primitives.BlockHeight(s.lastCommittedBlockHeight + 1),
			}).Build(),
			SignedTransactions: proposedTransactions.SignedTransactions,
		},
	}

	return blockPair, nil
}

func (s *service) leaderCollectVotesForBlock(blockPair *protocol.BlockPairContainer) (bool, error) {
	s.votesForActiveRound = make(chan bool)
	defer func() {
		close(s.votesForActiveRound)
		s.votesForActiveRound = nil
	}()

	_, err := s.gossip.SendLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
		Message: &gossipmessages.LeanHelixPrePrepareMessage{
			BlockPair: blockPair,
		},
	})
	if err != nil {
		return false, err
	}

	// asking for votes from everybody except ourselves
	for i := 0; i < int(s.config.NetworkSize(0))-1; i++ {
		<-s.votesForActiveRound
	}

	return true, nil
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
