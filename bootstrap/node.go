package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/consensus"
	"github.com/orbs-network/orbs-network-go/transactionpool"
)

type Node interface {
	SendTransaction(transaction *types.Transaction)
	CallMethod() int
}

type node struct {
	isLeader               bool
	gossip                 gossip.Gossip
	ledger                 ledger.Ledger
	events                 events.Events
	consensusAlgo          consensus.ConsensusAlgo
	transactionPool        transactionpool.TransactionPool
}

func NewNode(gossip gossip.Gossip,
	bp blockstorage.BlockPersistence,
	events events.Events,
	isLeader bool) Node {

	tp := transactionpool.NewTransactionPool(gossip)
	ledger := ledger.NewLedger(bp)
	consensusAlgo := consensus.NewConsensusAlgo(gossip, ledger, tp, events, isLeader)

	n := &node{
		isLeader:               isLeader,
		gossip:                 gossip,
		ledger:                 ledger,
		events:                 events,
		transactionPool:        tp,
		consensusAlgo:          consensusAlgo,
	}

	return n
}

func (n *node) SendTransaction(transaction *types.Transaction) {
	//TODO leader should also propagate transactions to other nodes
	if n.isLeader {
		n.transactionPool.Add(transaction)
	} else {
		n.gossip.ForwardTransaction(transaction)
	}
}

func (n *node) CallMethod() int {
	return n.ledger.GetState()
}
