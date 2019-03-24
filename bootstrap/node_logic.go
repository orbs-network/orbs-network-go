// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package bootstrap

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/servicesync"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/benchmarkconsensus"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum"
	ethereumAdapter "github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
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
	publicApi       services.PublicApi
	consensusAlgos  []services.ConsensusAlgo
	runtimeReporter interface{} // only needed so that the runtime reporter doesn't get GCed
}

func NewNodeLogic(
	ctx context.Context,
	gossipTransport gossipAdapter.Transport,
	blockPersistence blockStorageAdapter.BlockPersistence,
	statePersistence stateStorageAdapter.StatePersistence,
	stateBlockHeightReporter stateStorageAdapter.BlockHeightReporter,
	transactionPoolBlockHeightReporter transactionpool.BlockHeightReporter,
	nativeCompiler nativeProcessorAdapter.Compiler,
	logger log.BasicLogger,
	metricRegistry metric.Registry,
	nodeConfig config.NodeConfig,
	ethereumConnection ethereumAdapter.EthereumConnection,
) NodeLogic {

	config.NewValidator(logger).ValidateNodeLogic(nodeConfig)

	processors := make(map[protocol.ProcessorType]services.Processor)
	processors[protocol.PROCESSOR_TYPE_NATIVE] = native.NewNativeProcessor(nativeCompiler, nodeConfig, logger, metricRegistry)

	crosschainConnectors := make(map[protocol.CrosschainConnectorType]services.CrosschainConnector)
	crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM] = ethereum.NewEthereumCrosschainConnector(ethereumConnection, nodeConfig, logger, metricRegistry)

	gossipService := gossip.NewGossip(gossipTransport, nodeConfig, logger)
	stateStorageService := statestorage.NewStateStorage(nodeConfig, statePersistence, stateBlockHeightReporter, logger, metricRegistry)
	virtualMachineService := virtualmachine.NewVirtualMachine(stateStorageService, processors, crosschainConnectors, logger)
	transactionPoolService := transactionpool.NewTransactionPool(ctx, gossipService, virtualMachineService, transactionPoolBlockHeightReporter, nodeConfig, logger, metricRegistry)
	serviceSyncCommitters := []servicesync.BlockPairCommitter{servicesync.NewStateStorageCommitter(stateStorageService), servicesync.NewTxPoolCommitter(transactionPoolService)}
	blockStorageService := blockstorage.NewBlockStorage(ctx, nodeConfig, blockPersistence, gossipService, logger, metricRegistry, serviceSyncCommitters)
	publicApiService := publicapi.NewPublicApi(nodeConfig, transactionPoolService, virtualMachineService, blockStorageService, logger, metricRegistry)
	consensusContextService := consensuscontext.NewConsensusContext(transactionPoolService, virtualMachineService, stateStorageService, nodeConfig, logger, metricRegistry)

	benchmarkConsensusAlgo := benchmarkconsensus.NewBenchmarkConsensusAlgo(ctx, gossipService, blockStorageService, consensusContextService, logger, nodeConfig, metricRegistry)
	leanHelixAlgo := leanhelixconsensus.NewLeanHelixConsensusAlgo(ctx, gossipService, blockStorageService, consensusContextService, logger, nodeConfig, metricRegistry)

	consensusAlgos := make([]services.ConsensusAlgo, 0)
	consensusAlgos = append(consensusAlgos, benchmarkConsensusAlgo)
	consensusAlgos = append(consensusAlgos, leanHelixAlgo)

	metric.NewSystemReporter(ctx, metricRegistry, logger)
	runtimeReporter := metric.NewRuntimeReporter(ctx, metricRegistry, logger)
	metricRegistry.PeriodicallyReport(ctx, logger)

	ethereumConnection.ReportConnectionStatus(ctx, metricRegistry, logger)

	logger.Info("Node started")

	return &nodeLogic{
		publicApi:       publicApiService,
		consensusAlgos:  consensusAlgos,
		runtimeReporter: runtimeReporter,
	}
}

func (n *nodeLogic) PublicApi() services.PublicApi {
	return n.publicApi
}
