package consensus

import (
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"fmt"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossip"
	"sync"
)

type Config interface {
	NetworkSize(asOfBlock uint64) uint32
	NodeId() string
}

type ConsensusAlgo interface {
	gossip.LeanHelixConsensusHandler
}

type consensusAlgo struct {
	gossip          gossip.LeanHelixConsensus
	ledger          ledger.Ledger
	transactionPool services.TransactionPool
	events          instrumentation.Reporting
	loopControl     instrumentation.LoopControl

	votesForCurrentRound chan bool
	config               Config

	preparedBlock	[]byte
	commitCond 		*sync.Cond
}

func NewConsensusAlgo(gossip gossip.LeanHelixConsensus,
	ledger ledger.Ledger,
	transactionPool services.TransactionPool,
	events instrumentation.Reporting,
	loopControl instrumentation.LoopControl,
	config Config,
	isLeader bool) ConsensusAlgo {

	c := &consensusAlgo{
		gossip:          gossip,
		ledger:          ledger,
		transactionPool: transactionPool,
		events:          events,
		loopControl:     loopControl,
		config:          config,
		commitCond:		 sync.NewCond(&sync.Mutex{}),
	}

	gossip.RegisterLeanHelixConsensusHandler(c)

	if isLeader {
		go c.buildBlocksEventLoop()
	}

	return c
}

func (c *consensusAlgo) HandleLeanHelixPrePrepare(input *gossip.LeanHelixPrePrepareInput) (*gossip.LeanHelixOutput, error) {
	fmt.Printf("%s got pre-prepare\n", c.config.NodeId())
	c.preparedBlock = input.Block // each node will save this block
	return c.gossip.SendLeanHelixPrepare(&gossip.LeanHelixPrepareInput{})
}

func (c *consensusAlgo) HandleLeanHelixPrepare(input *gossip.LeanHelixPrepareInput) (*gossip.LeanHelixOutput, error) {
	// currently only leader should handle prepare
	if c.votesForCurrentRound != nil {
		c.events.Info(fmt.Sprintf("received vote"))
		c.votesForCurrentRound <- true
	}
	return nil, nil
}

func (c *consensusAlgo) HandleLeanHelixCommit(input *gossip.LeanHelixCommitInput) (*gossip.LeanHelixOutput, error) {
	fmt.Printf("%s committing block\n", c.config.NodeId())
	c.ledger.AddTransaction(protocol.SignedTransactionReader(c.preparedBlock))
	fmt.Printf("%s committed block\n", c.config.NodeId())
	c.preparedBlock = nil
	c.commitCond.Signal()
	return nil, nil
}

func (c *consensusAlgo) HandleLeanHelixViewChange(input *gossip.LeanHelixViewChangeInput) (*gossip.LeanHelixOutput, error) {
	panic("Not implemented")
}
func (c *consensusAlgo) HandleLeanHelixNewView(input *gossip.LeanHelixNewViewInput) (*gossip.LeanHelixOutput, error) {
	panic("Not implemented")
}

func (c *consensusAlgo) buildNextBlock(transaction *protocol.SignedTransaction) bool {
	votes, err := c.requestConsensusFor(transaction)
	if err != nil {
		c.events.Info(instrumentation.ConsensusError)
		return false
	}

	gotConsensus := true
	for i := uint32(0); i < c.config.NetworkSize(0); i++ {
		gotConsensus = gotConsensus && <- votes
	}

	if gotConsensus {
		if c.preparedBlock == nil {
			panic(fmt.Sprintf("Node [%s] is trying to commit a block that wasn't prepared", c.config.NodeId()))
		}
		c.gossip.SendLeanHelixCommit(&gossip.LeanHelixCommitInput{})
	}

	c.commitCond.Wait()

	close(c.votesForCurrentRound)

	return gotConsensus

}

func (c *consensusAlgo) buildBlocksEventLoop() {
	var currentBlock *protocol.SignedTransaction
	c.commitCond.L.Lock()

	c.loopControl.NewLoop("consensus_round", func() {

		if currentBlock == nil {
			res, _ := c.transactionPool.GetTransactionsForOrdering(&services.GetTransactionsForOrderingInput{MaxNumberOfTransactions: 1})
			currentBlock = res.SignedTransactions[0]
		}

		fmt.Printf("%s waiting for consensus\n", c.config.NodeId())
		if c.buildNextBlock(currentBlock) {
			fmt.Printf("%s got consensus\n", c.config.NodeId())
			currentBlock = nil
		}
	})
}

func (c *consensusAlgo) requestConsensusFor(transaction *protocol.SignedTransaction) (chan bool, error) {
	message := &gossip.LeanHelixPrePrepareInput{Block: transaction.Raw()}
	_, error := c.gossip.SendLeanHelixPrePrepare(message) //TODO send the actual input, not just a single tx bytes
	fmt.Printf("%s sent preprepare\n", c.config.NodeId())

	if error == nil {
		c.votesForCurrentRound = make(chan bool)

	} else {
		c.votesForCurrentRound = nil
	}

	return c.votesForCurrentRound, error

}
