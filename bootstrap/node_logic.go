package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/ledger"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelix"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/publicapi"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-network-go/gossip"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
)

type NodeLogic interface {
	GetPublicApi() services.PublicApi
}

type nodeLogic struct {
	isLeader        bool
	gossip          services.Gossip
	ledger          ledger.Ledger
	events          instrumentation.Reporting
	consensusAlgo   services.ConsensusAlgo
	transactionPool services.TransactionPool
	publicApi       services.PublicApi
}

func NewNodeLogic(
	gossipTransport gossip.Transport,
	bp blockStorageAdapter.BlockPersistence,
	events instrumentation.Reporting,
	loopControl instrumentation.LoopControl,
	nodeConfig config.NodeConfig,
	isLeader bool,
) NodeLogic {

	gossip := gossip.NewGossip(gossipTransport, nodeConfig)
	tp := transactionpool.NewTransactionPool(gossip)
	ledger := ledger.NewLedger(bp)
	consensusAlgo := leanhelix.NewConsensusAlgoLeanHelix(gossip, ledger, tp, events, loopControl, nodeConfig, isLeader)
	publicApi := publicapi.NewPublicApi(gossip, tp, ledger, events, isLeader)
	return &nodeLogic{
		publicApi:       publicApi,
		transactionPool: tp,
		consensusAlgo:   consensusAlgo,
	}
}

func (n *nodeLogic) GetPublicApi() services.PublicApi {
	return n.publicApi
}