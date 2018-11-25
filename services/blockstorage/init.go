package blockstorage

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter"
	blockSync "github.com/orbs-network/orbs-network-go/services/blockstorage/sync"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

const (
	// TODO extract it to the spec
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
	consensusBlocksHandlers []handlers.ConsensusBlocksHandler

	// lastCommittedBlock state variable is inside adapter.BlockPersistence (GetLastBlock)

	blockSync *blockSync.BlockSync

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

func NewBlockStorage(ctx context.Context, config config.BlockStorageConfig, persistence adapter.BlockPersistence, stateStorage services.StateStorage, gossip gossiptopics.BlockSync,
	txPool services.TransactionPool, parentLogger log.BasicLogger, metricFactory metric.Factory) services.BlockStorage {
	logger := parentLogger.WithTags(LogTag)

	s := &service{
		persistence:  persistence,
		stateStorage: stateStorage,
		gossip:       gossip,
		txPool:       txPool,
		logger:       logger,
		config:       config,
		metrics:      newMetrics(metricFactory),
	}

	gossip.RegisterBlockSyncHandler(s)
	s.blockSync = blockSync.NewBlockSync(ctx, config, gossip, s, logger, metricFactory)

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
	s.consensusBlocksHandlers = append(s.consensusBlocksHandlers, handler)

	// update the consensus algo about the latest block we have (for its initialization)
	s.UpdateConsensusAlgosAboutLatestCommittedBlock(context.TODO()) // TODO: (talkol) not sure if we should create a new context here or pass to RegisterConsensusBlocksHandler in code generation
}

