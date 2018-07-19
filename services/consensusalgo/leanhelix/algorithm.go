package leanhelix

import (
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

func leaderProposeNextBlock(
	transactionPool services.TransactionPool,
	lastCommittedBlockHeight primitives.BlockHeight,
) (*protocol.BlockPairContainer, error) {

	proposedTransactions, err := transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{
		MaxNumberOfTransactions: 1,
	})
	if err != nil {
		return nil, err
	}

	blockPair := &protocol.BlockPairContainer{
		TransactionsBlock: &protocol.TransactionsBlockContainer{
			Header: (&protocol.TransactionsBlockHeaderBuilder{
				ProtocolVersion: blockstorage.ProtocolVersion,
				BlockHeight:     primitives.BlockHeight(lastCommittedBlockHeight + 1),
			}).Build(),
			SignedTransactions: proposedTransactions.SignedTransactions,
		},
	}

	return blockPair, nil
}

func leaderCollectVotesForBlock(
	gossip gossiptopics.LeanHelix,
	votesForActiveRound *chan bool,
	blockPair *protocol.BlockPairContainer,
	networkSize int,
) (bool, error) {

	*votesForActiveRound = make(chan bool)
	defer func() {
		close(*votesForActiveRound)
		*votesForActiveRound = nil
	}()

	_, err := gossip.SendLeanHelixPrePrepare(&gossiptopics.LeanHelixPrePrepareInput{
		Message: &gossipmessages.LeanHelixPrePrepareMessage{
			BlockPair: blockPair,
		},
	})
	if err != nil {
		return false, nil
	}

	gotConsensus := true
	// asking for votes from everybody except ourselves
	for i := 0; i < networkSize-1; i++ {
		gotConsensus = gotConsensus && <-*votesForActiveRound
	}

	return gotConsensus, nil
}

func validatorVoteForNewBlockProposal(
	gossip gossiptopics.LeanHelix,
	blocksForRounds map[primitives.BlockHeight]*protocol.BlockPairContainer,
	blockPair *protocol.BlockPairContainer,
) error {

	blockHeight := blockPair.TransactionsBlock.Header.BlockHeight()
	blocksForRounds[blockHeight] = blockPair

	_, err := gossip.SendLeanHelixPrepare(&gossiptopics.LeanHelixPrepareInput{})
	return err

}

func leaderAddVote(votesForActiveRound *chan bool) {
	// TODO: we assume we only get votes for the active round, in the real world we can't assume this,
	// TODO:  but here since we don't move to the next round unless everybody voted, it's ok
	if *votesForActiveRound == nil {
		panic("received vote while not collecting votes")
	}
	*votesForActiveRound <- true
}

func commitBlockAndMoveToNextRound(
	blockStorage services.BlockStorage,
	blocksForRounds map[primitives.BlockHeight]*protocol.BlockPairContainer,
	lastCommittedBlockHeight primitives.BlockHeight,
) primitives.BlockHeight {

	blockPair, found := blocksForRounds[lastCommittedBlockHeight+1]
	if !found {
		panic("trying to commit a block that wasn't prepared")
	}

	blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPair,
	})

	delete(blocksForRounds, lastCommittedBlockHeight+1)
	return lastCommittedBlockHeight + 1

}
