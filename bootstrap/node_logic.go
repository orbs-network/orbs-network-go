package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelix"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type NodeLogic interface {
	PublicApi() services.PublicApi
}

type nodeLogic struct {
	events         instrumentation.Reporting
	publicApi      services.PublicApi
	consensusAlgos []services.ConsensusAlgo
}

func NewNodeLogic(
	ctx context.Context,
	gossipTransport gossipAdapter.Transport,
	blockPersistence blockStorageAdapter.BlockPersistence,
	statePersistence stateStorageAdapter.StatePersistence,
	reporting instrumentation.Reporting,
	nodeConfig config.NodeConfig,
) NodeLogic {

	gossip := gossip.NewGossip(gossipTransport, nodeConfig, reporting)
	transactionPool := transactionpool.NewTransactionPool(gossip, reporting)
	stateStorage := statestorage.NewStateStorage(statePersistence)
	blockStorage := blockstorage.NewBlockStorage(blockstorage.DefaultBlockStorageConfig(), blockPersistence, stateStorage, reporting)
	nativeProcessor := native.NewNativeProcessor()
	ethereumCrosschainConnector := ethereum.NewEthereumCrosschainConnector()
	virtualMachine := virtualmachine.NewVirtualMachine(blockStorage, stateStorage, nativeProcessor, ethereumCrosschainConnector)
	publicApi := publicapi.NewPublicApi(transactionPool, virtualMachine, reporting)
	consensusContext := consensuscontext.NewConsensusContext(transactionPool, virtualMachine, nil)

	var consensusAlgos []services.ConsensusAlgo
	consensusAlgos = append(consensusAlgos, leanhelix.NewLeanHelixConsensusAlgo(gossip, blockStorage, transactionPool, consensusContext, reporting, nodeConfig))
	consensusAlgos = append(consensusAlgos, benchmarkconsensus.NewBenchmarkConsensusAlgo(ctx, gossip, blockStorage, consensusContext, reporting, nodeConfig))

	return &nodeLogic{
		publicApi:      publicApi,
		consensusAlgos: consensusAlgos,
	}
}

func (n *nodeLogic) PublicApi() services.PublicApi {
	return n.publicApi
}
