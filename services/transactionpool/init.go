package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

func NewTransactionPool(ctx context.Context,
	gossip gossiptopics.TransactionRelay,
	virtualMachine services.VirtualMachine,
	blockHeightReporter BlockHeightReporter,
	config config.TransactionPoolConfig,
	parent log.BasicLogger,
	metricFactory metric.Factory) services.TransactionPool {

	if blockHeightReporter == nil {
		blockHeightReporter = synchronization.NopHeightReporter{}
	}
	waiter := newTransactionWaiter()
	onNewTransaction := func() { waiter.inc(ctx) }
	pendingPool := NewPendingPool(config.TransactionPoolPendingPoolSizeInBytes, metricFactory, onNewTransaction)
	committedPool := NewCommittedPool(metricFactory)

	logger := parent.WithTags(LogTag)

	txForwarder := NewTransactionForwarder(ctx, logger, config, gossip)

	s := &service{
		gossip:         gossip,
		virtualMachine: virtualMachine,
		config:         config,
		logger:         logger,

		pendingPool:          pendingPool,
		committedPool:        committedPool,
		blockTracker:         synchronization.NewBlockTracker(logger, 0, uint16(config.BlockTrackerGraceDistance())),
		blockHeightReporter:  blockHeightReporter,
		transactionForwarder: txForwarder,
		transactionWaiter:    waiter,
	}

	s.mu.lastCommittedBlockTimestamp = primitives.TimestampNano(0) // this is so that we reject transactions on startup, before any block has been committed
	s.metrics.blockHeight = metricFactory.NewGauge("TransactionPool.BlockHeight")

	gossip.RegisterTransactionRelayHandler(s)
	pendingPool.onTransactionRemoved = s.onTransactionError

	startCleaningProcess(ctx, config.TransactionPoolCommittedPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.committedPool, logger)
	startCleaningProcess(ctx, config.TransactionPoolPendingPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.pendingPool, logger)

	return s
}

func (s *service) onTransactionError(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) {
	bh, ts := s.lastCommittedBlockHeightAndTime()
	if removalReason != protocol.TRANSACTION_STATUS_COMMITTED {
		for _, trh := range s.transactionResultsHandlers {
			_, err := trh.HandleTransactionError(ctx, &handlers.HandleTransactionErrorInput{
				Txhash:            txHash,
				TransactionStatus: removalReason,
				BlockTimestamp:    ts,
				BlockHeight:       bh,
			})
			if err != nil {
				s.logger.Info("notify tx error failed", log.Error(err))
			}
		}
	}
}
