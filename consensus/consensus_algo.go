package consensus

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/transactionpool"
)

type ConsensusAlgo interface {
	gossip.ConsensusListener
}

type consensusAlgo struct {
	gossip          gossip.Gossip
	ledger          ledger.Ledger
	transactionPool transactionpool.TransactionPool
	events          events.Events
}

func NewConsensusAlgo(gossip gossip.Gossip,
											ledger ledger.Ledger,
											transactionPool transactionpool.TransactionPool,
	                    events events.Events,
	                    isLeader bool) ConsensusAlgo {

	c := &consensusAlgo{
		gossip:          gossip,
		ledger:          ledger,
		transactionPool: transactionPool,
		events:          events,
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

func (c *consensusAlgo) ValidateConsensusFor(transaction *types.Transaction) bool {
	return true
}

func (c *consensusAlgo) buildNextBlock(transaction *types.Transaction) bool {
	gotConsensus, err := c.gossip.HasConsensusFor(transaction)

	if err != nil {
		return false
	}

	if gotConsensus {
		c.gossip.CommitTransaction(transaction)
	}

	return gotConsensus

}

func (c *consensusAlgo) buildBlocksEventLoop() {
	var t *types.Transaction
	for {
		if t == nil {
			t = c.transactionPool.Next()
		}

		if c.buildNextBlock(t) {
			t = nil
		}
		c.events.FinishedConsensusRound()
	}
}
