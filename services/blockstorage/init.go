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
	s.UpdateConsensusAlgosAboutLatestCommittedBlock(context.TODO()) // TODO: (talkol) not sure if we should create a new context here or pass to RegisterConsensusBlocksHandler in code generation
}

func (s *service) appendHandlerUnderLock(handler handlers.ConsensusBlocksHandler) {
	s.consensusBlocksHandlers.Lock()
	defer s.consensusBlocksHandlers.Unlock()
	s.consensusBlocksHandlers.handlers = append(s.consensusBlocksHandlers.handlers, handler)
}
