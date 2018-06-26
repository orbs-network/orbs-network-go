package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/blockstorage"
)

type Node interface {
	gossip.Listener
	SendTransaction(transaction *types.Transaction)
	CallMethod() int
}

type node struct {
	isLeader               bool
	gossip                 gossip.Gossip
	blockPersistence       blockstorage.BlockPersistence
	ledger                 ledger.Ledger
	pendingTransactionPool chan *types.Transaction
}

func NewNode(gossip gossip.Gossip, bp blockstorage.BlockPersistence, isLeader bool) Node {
	n := &node{
		isLeader:               isLeader,
		gossip:                 gossip,
		blockPersistence:       bp,
		ledger:                 ledger.NewLedger(),
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

func (n *node) buildNextBlock(transaction *types.Transaction) {
	if n.gossip.HasConsensusFor(transaction) {
		n.gossip.CommitTransaction(transaction)
	}
}

func (n *node) buildBlocksEventLoop() {
	t := <-n.pendingTransactionPool
	n.buildNextBlock(t)
}
