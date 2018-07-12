package bootstrap

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelix"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type NodeLogic interface {
	GetPublicApi() services.PublicApi
}

type nodeLogic struct {
	isLeader  bool
	events    instrumentation.Reporting
	leanHelix services.ConsensusAlgo // TODO: change this to a map
	publicApi services.PublicApi
}

func NewNodeLogic(
	gossipTransport gossipAdapter.Transport,
	blockPersistence blockStorageAdapter.BlockPersistence,
	events instrumentation.Reporting,
	loopControl instrumentation.LoopControl,
	nodeConfig config.NodeConfig,
	isLeader bool,
) NodeLogic {

	gossip := gossip.NewGossip(gossipTransport, nodeConfig)
	transactionPool := transactionpool.NewTransactionPool(gossip)
	blockStorage := blockstorage.NewBlockStorage(blockPersistence)
	nativeProcessor := native.NewNativeProcessor()
	ethereumCrosschainConnector := ethereum.NewEthereumCrosschainConnector()
	virtualMachine := virtualmachine.NewVirtualMachine(blockStorage, nativeProcessor, ethereumCrosschainConnector, blockPersistence)
	publicApi := publicapi.NewPublicApi(transactionPool, virtualMachine, events, isLeader)
	consensusContext := consensuscontext.NewConsensusContext(transactionPool, virtualMachine, nil)
	leanHelixConsensusAlgo := leanhelix.NewLeanHelixConsensusAlgo(gossip, blockStorage, transactionPool, consensusContext, events, loopControl, nodeConfig, isLeader)

	return &nodeLogic{
		publicApi: publicApi,
		leanHelix: leanHelixConsensusAlgo,
	}
}

func (n *nodeLogic) GetPublicApi() services.PublicApi {
	return n.publicApi
}
