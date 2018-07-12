package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelix"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type NodeLogic interface {
	GetPublicApi() services.PublicApi
}

type nodeLogic struct {
	isLeader        bool
	gossip          services.Gossip
	blockStorage    services.BlockStorage
	virtualMachine  services.VirtualMachine
	events          instrumentation.Reporting
	consensusAlgo   services.ConsensusAlgo
	transactionPool services.TransactionPool
	publicApi       services.PublicApi
}

func NewNodeLogic(
	gossipTransport gossipAdapter.Transport,
	bp blockStorageAdapter.BlockPersistence,
	events instrumentation.Reporting,
	loopControl instrumentation.LoopControl,
	nodeConfig config.NodeConfig,
	isLeader bool,
) NodeLogic {

	gossip := gossip.NewGossip(gossipTransport, nodeConfig)
	tp := transactionpool.NewTransactionPool(gossip)
	blockStorage := blockstorage.NewBlockStorage(bp)
	virtualMachine := virtualmachine.NewVirtualMachine(blockStorage, bp)
	consensusAlgo := leanhelix.NewConsensusAlgoLeanHelix(gossip, blockStorage, tp, events, loopControl, nodeConfig, isLeader)
	publicApi := publicapi.NewPublicApi(tp, virtualMachine, events, isLeader)
	return &nodeLogic{
		publicApi:       publicApi,
		transactionPool: tp,
		consensusAlgo:   consensusAlgo,
	}
}

func (n *nodeLogic) GetPublicApi() services.PublicApi {
	return n.publicApi
}
