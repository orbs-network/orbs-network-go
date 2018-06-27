package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/consensus"
)

type Node interface {
	gossip.TransactionListener
	SendTransaction(transaction *types.Transaction)
	CallMethod() int
}

type node struct {
	isLeader               bool
	gossip                 gossip.Gossip
	ledger                 ledger.Ledger
	pendingTransactionPool chan *types.Transaction
	events                 events.Events
	consensusAlgo          consensus.ConsensusAlgo
}

func NewNode(gossip gossip.Gossip,
	bp blockstorage.BlockPersistence,
	events events.Events,
	isLeader bool) Node {

	tp := make(chan *types.Transaction, 10)
	ledger := ledger.NewLedger(bp)
	consensusAlgo := consensus.NewConsensusAlgo(gossip, ledger, tp, events)

	n := &node{
		isLeader:               isLeader,
		gossip:                 gossip,
		ledger:                 ledger,
		events:                 events,
		pendingTransactionPool: tp,
		consensusAlgo:          consensusAlgo,
	}

	gossip.RegisterTransactionListener(n)

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
