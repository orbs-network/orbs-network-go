package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/events"
)

type Node interface {
	gossip.Listener
	SendTransaction(transaction *types.Transaction)
	CallMethod() int
}

type node struct {
	isLeader               bool
	gossip                 gossip.Gossip
	ledger                 ledger.Ledger
	pendingTransactionPool chan *types.Transaction
	events                 events.Events
}

func NewNode(gossip gossip.Gossip,
	bp blockstorage.BlockPersistence,
	e events.Events,
	isLeader bool) Node {

	n := &node{
		isLeader:               isLeader,
		gossip:                 gossip,
		ledger:                 ledger.NewLedger(bp),
		events:                 e,
		pendingTransactionPool: make(chan *types.Transaction, 10),
	}

	go n.buildBlocksEventLoop()
	return n
}

func (n *node) SendTransaction(transaction *types.Transaction) {
	if n.isLeader {
		n.pendingTransactionPool <- transaction
	} else {
		n.gossip.ForwardTransaction(transaction)
	}
}

func (n *node) CallMethod() int {
	return n.ledger.GetState()
}

func (n *node) OnForwardTransaction(transaction *types.Transaction) {
	if n.isLeader {
		n.pendingTransactionPool <- transaction
	}
}

func (n *node) OnCommitTransaction(transaction *types.Transaction) {
	n.ledger.AddTransaction(transaction)
}

func (n *node) ValidateConsensusFor(transaction *types.Transaction) bool {
	return true
}

func (n *node) buildNextBlock(transaction *types.Transaction) bool {
	gotConsensus, err := n.gossip.HasConsensusFor(transaction)

	if err != nil {
		return false
	}

	if gotConsensus {
		n.gossip.CommitTransaction(transaction)
	}

	return gotConsensus

}

func (n *node) buildBlocksEventLoop() {
	var t *types.Transaction
	for {
		if t == nil {
			t = <- n.pendingTransactionPool
		}

		if n.buildNextBlock(t) {
			t = nil
		}
		n.events.FinishedConsensusRound()
	}
}
