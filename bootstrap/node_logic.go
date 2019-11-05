// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package bootstrap

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/signer"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
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
	txPoolAdapter "github.com/orbs-network/orbs-network-go/services/transactionpool/adapter"
	"github.com/orbs-network/orbs-network-go/services/virtualmachine"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

type NodeLogic interface {
	govnr.ShutdownWaiter
	PublicApi() services.PublicApi
}

type nodeLogic struct {
	govnr.TreeSupervisor
	publicApi      services.PublicApi
	consensusAlgos []services.ConsensusAlgo
}

func NewNodeLogic(
	parentCtx context.Context,
	gossipTransport gossipAdapter.Transport,
	blockPersistence blockStorageAdapter.BlockPersistence,
	statePersistence stateStorageAdapter.StatePersistence,
	stateBlockHeightReporter stateStorageAdapter.BlockHeightReporter,
	transactionPoolBlockHeightReporter transactionpool.BlockHeightReporter,
	maybeClock txPoolAdapter.Clock,
	nativeCompiler nativeProcessorAdapter.Compiler,
	logger log.Logger,
	metricRegistry metric.Registry,
	nodeConfig config.NodeConfig,
	ethereumConnection ethereumAdapter.EthereumConnection,
) NodeLogic {

	ctx := trace.ContextWithNodeId(parentCtx, nodeConfig.NodeAddress().String())

	config.NewValidator(logger).ValidateNodeLogic(nodeConfig)

	processors := make(map[protocol.ProcessorType]services.Processor)
	processors[protocol.PROCESSOR_TYPE_NATIVE] = native.NewNativeProcessor(nativeCompiler, nodeConfig, logger, metricRegistry)
	addExtraProcessors(processors, nodeConfig, logger)

	crosschainConnectors := make(map[protocol.CrosschainConnectorType]services.CrosschainConnector)
	crosschainConnectors[protocol.CROSSCHAIN_CONNECTOR_TYPE_ETHEREUM] = ethereum.NewEthereumCrosschainConnector(ethereumConnection, nodeConfig, logger, metricRegistry)

	signer, err := signer.New(nodeConfig)
	if err != nil {
		panic(fmt.Sprintf("could not instantiate NodeLogic: %s", err))
	}

	gossipService := gossip.NewGossip(ctx, gossipTransport, nodeConfig, logger, metricRegistry)
	stateStorageService := statestorage.NewStateStorage(nodeConfig, statePersistence, stateBlockHeightReporter, logger, metricRegistry)
	virtualMachineService := virtualmachine.NewVirtualMachine(stateStorageService, processors, crosschainConnectors, logger)
	transactionPoolService := transactionpool.NewTransactionPool(ctx, maybeClock, gossipService, virtualMachineService, signer, transactionPoolBlockHeightReporter, nodeConfig, logger, metricRegistry)
	serviceSyncCommitters := []servicesync.BlockPairCommitter{servicesync.NewStateStorageCommitter(stateStorageService), servicesync.NewTxPoolCommitter(transactionPoolService)}
	blockStorageService := blockstorage.NewBlockStorage(ctx, nodeConfig, blockPersistence, gossipService, logger, metricRegistry, serviceSyncCommitters)
	publicApiService := publicapi.NewPublicApi(nodeConfig, transactionPoolService, virtualMachineService, blockStorageService, logger, metricRegistry)
	consensusContextService := consensuscontext.NewConsensusContext(transactionPoolService, virtualMachineService, stateStorageService, nodeConfig, logger, metricRegistry)

	benchmarkConsensusAlgo := benchmarkconsensus.NewBenchmarkConsensusAlgo(ctx, gossipService, blockStorageService, consensusContextService, signer, logger, nodeConfig, metricRegistry)
	leanHelixAlgo := leanhelixconsensus.NewLeanHelixConsensusAlgo(ctx, gossipService, blockStorageService, consensusContextService, signer, logger, nodeConfig, metricRegistry)

	consensusAlgos := make([]services.ConsensusAlgo, 0)
	consensusAlgos = append(consensusAlgos, benchmarkConsensusAlgo)
	consensusAlgos = append(consensusAlgos, leanHelixAlgo)

	metric.RegisterConfigIndicators(metricRegistry, nodeConfig)

	logger.Info("Node started")

	node := &nodeLogic{
		publicApi:      publicApiService,
		consensusAlgos: consensusAlgos,
	}

	node.Supervise(gossipService)
	node.Supervise(blockStorageService)
	node.Supervise(benchmarkConsensusAlgo)
	node.Supervise(leanHelixAlgo)
	node.Supervise(metric.NewSystemReporter(ctx, metricRegistry, logger))
	node.Supervise(metric.NewRuntimeReporter(ctx, metricRegistry, logger))
	node.Supervise(metricRegistry.PeriodicallyRotate(ctx, logger))
	if nodeConfig.NTPEndpoint() != "" {
		node.Supervise(metric.NewNtpReporter(ctx, metricRegistry, logger, nodeConfig.NTPEndpoint()))
	}

	return node
}

func (n *nodeLogic) PublicApi() services.PublicApi {
	return n.publicApi
}
