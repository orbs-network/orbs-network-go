package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	nativeProcessorAdapter "github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
	"github.com/orbs-network/orbs-network-go/services/statestorage"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type NodeLogic interface {
	PublicApi() services.PublicApi
}

type nodeLogic struct {
	publicApi      services.PublicApi
	consensusAlgos []services.ConsensusAlgo
}

func NewNodeLogic(
	ctx context.Context,
	gossipTransport gossipAdapter.Transport,
	blockPersistence blockStorageAdapter.BlockPersistence,
	statePersistence stateStorageAdapter.StatePersistence,
	nativeCompiler nativeProcessorAdapter.Compiler,
	logger log.BasicLogger,
	metricFactory metric.Factory,
	nodeConfig config.NodeConfig,
) NodeLogic {

	processors := make(map[protocol.ProcessorType]services.Processor)
	processors[protocol.PROCESSOR_TYPE_NATIVE] = native.NewNativeProcessor(nativeCompiler, logger)

	crosschainConnectors := make(map[protocol.CrosschainConnectorType]services.CrosschainConnector)
	crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM] = ethereum.NewEthereumCrosschainConnector()

	gossipService := gossip.NewGossip(gossipTransport, nodeConfig, logger)
	stateStorageService := statestorage.NewStateStorage(nodeConfig, statePersistence, logger)
	virtualMachineService := virtualmachine.NewVirtualMachine(stateStorageService, processors, crosschainConnectors, logger)
	transactionPoolService := transactionpool.NewTransactionPool(ctx, gossipService, virtualMachineService, nodeConfig, logger, metricFactory)
	blockStorageService := blockstorage.NewBlockStorage(ctx, nodeConfig, blockPersistence, stateStorageService, gossipService, transactionPoolService, logger)
	publicApiService := publicapi.NewPublicApi(ctx, nodeConfig, transactionPoolService, virtualMachineService, blockStorageService, logger, metricFactory)
	consensusContextService := consensuscontext.NewConsensusContext(transactionPoolService, virtualMachineService, nil, nodeConfig, logger, metricFactory)

	consensusAlgos := make([]services.ConsensusAlgo, 0)

	// TODO: Restore this when lean-helix-go submodule is integrated
	//consensusAlgos = append(consensusAlgos, leanhelix.NewLeanHelixConsensusAlgo(gossipService, blockStorageService, transactionPoolService, consensusContextService, logger, nodeConfig))
	consensusAlgos = append(consensusAlgos, benchmarkconsensus.NewBenchmarkConsensusAlgo(ctx, gossipService, blockStorageService, consensusContextService, logger, nodeConfig))

	return &nodeLogic{
		publicApi:      publicApiService,
		consensusAlgos: consensusAlgos,
	}
}

func (n *nodeLogic) PublicApi() services.PublicApi {
	return n.publicApi
}
