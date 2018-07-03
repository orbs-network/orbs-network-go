package consensus

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/transactionpool"
	"github.com/orbs-network/orbs-network-go/loopcontrol"
)

const numOfRequiredVotes = 2

type ConsensusAlgo interface {
	gossip.ConsensusListener
}

type consensusAlgo struct {
	gossip          gossip.Gossip
	ledger          ledger.Ledger
	transactionPool transactionpool.TransactionPool
	events          events.Events
	loopControl     loopcontrol.LoopControl

	votesForCurrentRound chan bool
}

func NewConsensusAlgo(gossip gossip.Gossip,
	ledger ledger.Ledger,
	transactionPool transactionpool.TransactionPool,
	events events.Events,
	loopControl loopcontrol.LoopControl,
	isLeader bool) ConsensusAlgo {

	c := &consensusAlgo{
		gossip:          gossip,
		ledger:          ledger,
		transactionPool: transactionPool,
		events:          events,
		loopControl:     loopControl,
	}

	gossip.RegisterConsensusListener(c)

	if isLeader {
		go c.buildBlocksEventLoop()
	}

	return c
}

func (c *consensusAlgo) OnCommitTransaction(transaction *types.Transaction) {
	c.ledger.AddTransaction(transaction)
}

func (c *consensusAlgo) OnVote(yay bool) {
	if c.votesForCurrentRound != nil { //TODO remove if
		c.votesForCurrentRound <- yay
	}
}

func (c *consensusAlgo) OnVoteRequest(transaction *types.Transaction) {
	c.gossip.BroadcastVote(true)
}

func (c *consensusAlgo) buildNextBlock(transaction *types.Transaction) bool {
	votes, err := c.requestConsensusFor(transaction)
	if err != nil {
		c.events.Report(events.ConsensusError)
		return false
	}

	gotConsensus := true
	for i := 0 ; i < numOfRequiredVotes; i++ {
		gotConsensus = gotConsensus && <- votes
	}

	close(c.votesForCurrentRound)

	if gotConsensus {
		c.gossip.CommitTransaction(transaction)
	}

	return gotConsensus

}

func (c *consensusAlgo) buildBlocksEventLoop() {
	var currentBlock *types.Transaction

	c.loopControl.NewLoop("consensus_round", func() {

		if currentBlock == nil {
			currentBlock = c.transactionPool.Next()
		}

		if c.buildNextBlock(currentBlock) {
			currentBlock = nil
		}
	})
}

func (c *consensusAlgo) requestConsensusFor(transaction *types.Transaction) (chan bool, error) {
	error := c.gossip.RequestConsensusFor(transaction)

	if error == nil {
		c.votesForCurrentRound = make(chan bool)

	} else {
		c.votesForCurrentRound = nil
	}

	return c.votesForCurrentRound, error


}
