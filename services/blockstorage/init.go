// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/servicesync"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
)

const (
	ProtocolVersion = primitives.ProtocolVersion(1)
)

var LogTag = log.Service("block-storage")

type service struct {
	persistence  adapter.BlockPersistence
	stateStorage services.StateStorage
	gossip       gossiptopics.BlockSync
	txPool       services.TransactionPool

	config config.BlockStorageConfig

	logger                  log.BasicLogger
	consensusBlocksHandlers struct {
		sync.RWMutex
		handlers []handlers.ConsensusBlocksHandler
	}

	// lastCommittedBlock state variable is inside adapter.BlockPersistence (GetLastBlock)

	nodeSync *internodesync.BlockSync

	metrics *metrics
}

type metrics struct {
	blockHeight *metric.Gauge
}

func newMetrics(m metric.Factory) *metrics {
	return &metrics{
		blockHeight: m.NewGauge("BlockStorage.BlockHeight"),
	}
}

func NewBlockStorage(ctx context.Context, config config.BlockStorageConfig, persistence adapter.BlockPersistence, gossip gossiptopics.BlockSync,
	parentLogger log.BasicLogger, metricFactory metric.Factory, blockPairReceivers []servicesync.BlockPairCommitter) services.BlockStorage {
	logger := parentLogger.WithTags(LogTag)

	s := &service{
		persistence: persistence,
		gossip:      gossip,
		logger:      logger,
		config:      config,
		metrics:     newMetrics(metricFactory),
	}

	gossip.RegisterBlockSyncHandler(s)
	s.nodeSync = internodesync.NewBlockSync(ctx, config, gossip, s, logger, metricFactory)

	for _, bpr := range blockPairReceivers {
		servicesync.NewServiceBlockSync(ctx, logger, persistence, bpr)
	}

	height, err := persistence.GetLastBlockHeight()
	if err != nil {
		logger.Error("could not read block height from adapter", log.Error(err))
	}
	s.metrics.blockHeight.Update(int64(height))

	return s
}

func getBlockHeight(block *protocol.BlockPairContainer) primitives.BlockHeight {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.BlockHeight()
}

func getBlockTimestamp(block *protocol.BlockPairContainer) primitives.TimestampNano {
	if block == nil {
		return 0
	}
	return block.TransactionsBlock.Header.Timestamp()
}

func (s *service) RegisterConsensusBlocksHandler(handler handlers.ConsensusBlocksHandler) {
	s.appendHandlerUnderLock(handler)
	// update the consensus algo about the latest block we have (for its initialization)
	s.UpdateConsensusAlgosAboutLastCommittedBlockInLocalPersistence(context.TODO())
}

func (s *service) appendHandlerUnderLock(handler handlers.ConsensusBlocksHandler) {
	s.consensusBlocksHandlers.Lock()
	defer s.consensusBlocksHandlers.Unlock()
	s.consensusBlocksHandlers.handlers = append(s.consensusBlocksHandlers.handlers, handler)
}
