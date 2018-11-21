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
	"time"
)

func NewTransactionPool(ctx context.Context,
	gossip gossiptopics.TransactionRelay,
	virtualMachine services.VirtualMachine,
	config config.TransactionPoolConfig,
	parent log.BasicLogger,
	metricFactory metric.Factory) services.TransactionPool {

	pendingPool := NewPendingPool(config.TransactionPoolPendingPoolSizeInBytes, metricFactory)
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
		blockTracker:         synchronization.NewBlockTracker(0, uint16(config.BlockTrackerGraceDistance())),
		transactionForwarder: txForwarder,
	}

	s.mu.lastCommittedBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) // this is so that we do not reject transactions on startup, before any block has been committed

	gossip.RegisterTransactionRelayHandler(s)
	pendingPool.onTransactionRemoved = s.onTransactionError

	startCleaningProcess(ctx, config.TransactionPoolCommittedPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.committedPool, logger)
	startCleaningProcess(ctx, config.TransactionPoolPendingPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.pendingPool, logger)

	return s
}

func (s *service) onTransactionError(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) {
	bh, ts := s.currentBlockHeightAndTime()
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
