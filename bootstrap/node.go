package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/consensus"
	"github.com/orbs-network/orbs-network-go/transactionpool"
	"github.com/orbs-network/orbs-network-go/publicapi"
	"github.com/orbs-network/orbs-network-go/config"
)

type Node interface {
	GetPublicApi() publicapi.PublicApi
}

type node struct {
	isLeader        bool
	gossip          gossip.Gossip
	ledger          ledger.Ledger
	events          instrumentation.Reporting
	consensusAlgo   consensus.ConsensusAlgo
	transactionPool transactionpool.TransactionPool
	publicApi       publicapi.PublicApi
}

func NewNode(gossipTransport gossip.Transport,
	bp blockstorage.BlockPersistence,
	events instrumentation.Reporting,
	loopControl instrumentation.LoopControl,
	nodeConfig config.NodeConfig,
	isLeader bool) Node {

	gossip := gossip.NewGossip(gossipTransport)
	tp := transactionpool.NewTransactionPool(gossip)
	ledger := ledger.NewLedger(bp)
	consensusAlgo := consensus.NewConsensusAlgo(gossip, ledger, tp, events, loopControl, nodeConfig, isLeader)
	publicApi := publicapi.NewPublicApi(gossip, tp, ledger, events, isLeader)

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
