package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/consensus"
	"github.com/orbs-network/orbs-network-go/transactionpool"
	"github.com/orbs-network/orbs-network-go/publicapi"
)

type Node interface {
	GetPublicApi() publicapi.PublicApi
}

type node struct {
	isLeader        bool
	gossip          gossip.Gossip
	ledger          ledger.Ledger
	events          events.Events
	consensusAlgo   consensus.ConsensusAlgo
	transactionPool transactionpool.TransactionPool
	publicApi       publicapi.PublicApi
}

func NewNode(gossip gossip.Gossip,
	bp blockstorage.BlockPersistence,
	events events.Events,
	isLeader bool) Node {

	tp := transactionpool.NewTransactionPool(gossip)
	ledger := ledger.NewLedger(bp)
	consensusAlgo := consensus.NewConsensusAlgo(gossip, ledger, tp, events, isLeader)
	publicApi := publicapi.NewPublicApi(gossip, tp, ledger, isLeader)

	n := &node{
		publicApi:       publicApi,
		transactionPool: tp,
		consensusAlgo:   consensusAlgo,
	}

	return n
}

func (n *node) GetPublicApi() publicapi.PublicApi {
	return n.publicApi
}
