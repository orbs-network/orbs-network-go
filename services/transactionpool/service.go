package transactionpool

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"sync"
	"time"
)

var LogTag = log.Service("transaction-pool")

type service struct {
	gossip                     gossiptopics.TransactionRelay
	virtualMachine             services.VirtualMachine
	transactionResultsHandlers []handlers.TransactionResultsHandler
	logger                     log.BasicLogger
	config                     config.TransactionPoolConfig

	mu struct {
		sync.RWMutex
		lastCommittedBlockHeight    primitives.BlockHeight
		lastCommittedBlockTimestamp primitives.TimestampNano
	}

	pendingPool          *pendingTxPool
	committedPool        *committedTxPool
	blockTracker         *synchronization.BlockTracker
	transactionForwarder *transactionForwarder
}

func NewTransactionPool(ctx context.Context,
	gossip gossiptopics.TransactionRelay,
	virtualMachine services.VirtualMachine,
	config config.TransactionPoolConfig,
	logger log.BasicLogger,
	metricFactory metric.Factory) services.TransactionPool {

	pendingPool := NewPendingPool(config.TransactionPoolPendingPoolSizeInBytes, metricFactory)
	committedPool := NewCommittedPool(metricFactory)

	txForwarder := NewTransactionForwarder(ctx, logger, config, gossip)

	s := &service{
		gossip:         gossip,
		virtualMachine: virtualMachine,
		config:         config,
		logger:         logger.WithTags(LogTag),

		pendingPool:          pendingPool,
		committedPool:        committedPool,
		blockTracker:         synchronization.NewBlockTracker(0, uint16(config.BlockTrackerGraceDistance()), time.Duration(config.BlockTrackerGraceTimeout())),
		transactionForwarder: txForwarder,
	}

	s.mu.lastCommittedBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano()) // this is so that we do not reject transactions on startup, before any block has been committed

	gossip.RegisterTransactionRelayHandler(s)
	pendingPool.onTransactionRemoved = s.onTransactionError

	startCleaningProcess(ctx, config.TransactionPoolCommittedPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.committedPool, logger)
	startCleaningProcess(ctx, config.TransactionPoolPendingPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.pendingPool, logger)

	return s
}

func (s *service) GetCommittedTransactionReceipt(ctx context.Context, input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {

	if input.TransactionTimestamp > s.currentNodeTimeWithGrace() {
		return s.getTxResult(nil, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_AHEAD_OF_NODE_TIME), nil
	}

	if tx := s.pendingPool.get(input.Txhash); tx != nil {
		return s.getTxResult(nil, protocol.TRANSACTION_STATUS_PENDING), nil
	}

	if tx := s.committedPool.get(input.Txhash); tx != nil {
		return s.getTxResult(tx.receipt, protocol.TRANSACTION_STATUS_COMMITTED), nil
	}

	return s.getTxResult(nil, protocol.TRANSACTION_STATUS_NO_RECORD_FOUND), nil
}

func (s *service) currentNodeTimeWithGrace() primitives.TimestampNano {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.lastCommittedBlockTimestamp + primitives.TimestampNano(s.config.TransactionPoolFutureTimestampGraceTimeout().Nanoseconds())
}

func (s *service) currentBlockHeightAndTime() (primitives.BlockHeight, primitives.TimestampNano) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.lastCommittedBlockHeight, s.mu.lastCommittedBlockTimestamp
}

func (s *service) ValidateTransactionsForOrdering(ctx context.Context, input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	if err := s.blockTracker.WaitForBlock(ctx, input.BlockHeight); err != nil {
		return nil, err
	}

	vctx := s.createValidationContext()

	for _, tx := range input.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		if s.committedPool.has(txHash) {
			return nil, errors.Errorf("transaction with hash %s already committed", txHash)
		}

		if err := vctx.validateTransaction(tx); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("transaction with hash %s is invalid", txHash))
		}
	}

	//TODO handle error from vm
	bh, _ := s.currentBlockHeightAndTime()
	preOrderResults, _ := s.virtualMachine.TransactionSetPreOrder(ctx, &services.TransactionSetPreOrderInput{
		SignedTransactions: input.SignedTransactions,
		BlockHeight:        bh,
	})

	for i, tx := range input.SignedTransactions {
		if status := preOrderResults.PreOrderResults[i]; status != protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
			return nil, errors.Errorf("transaction with hash %s failed pre-order checks with status %s", digest.CalcTxHash(tx.Transaction()), status)
		}
	}
	return &services.ValidateTransactionsForOrderingOutput{}, nil
}

func (s *service) createValidationContext() *validationContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.mu.lastCommittedBlockTimestamp == 0 {
		panic("last committed block timestamp should never be zero!")
	}
	return &validationContext{
		expiryWindow:                s.config.TransactionPoolTransactionExpirationWindow(),
		lastCommittedBlockTimestamp: s.mu.lastCommittedBlockTimestamp,
		futureTimestampGrace:        s.config.TransactionPoolFutureTimestampGraceTimeout(),
		virtualChainId:              s.config.VirtualChainId(),
	}
}

func (s *service) getTxResult(receipt *protocol.TransactionReceipt, status protocol.TransactionStatus) *services.GetCommittedTransactionReceiptOutput {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &services.GetCommittedTransactionReceiptOutput{
		TransactionStatus:  status,
		TransactionReceipt: receipt,
		BlockHeight:        s.mu.lastCommittedBlockHeight,
		BlockTimestamp:     s.mu.lastCommittedBlockTimestamp,
	}
}

func (s *service) onTransactionError(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) {
	bh, ts := s.currentBlockHeightAndTime()
	if removalReason != protocol.TRANSACTION_STATUS_COMMITTED {
		for _, trh := range s.transactionResultsHandlers {
			trh.HandleTransactionError(ctx, &handlers.HandleTransactionErrorInput{
				Txhash:            txHash,
				TransactionStatus: removalReason,
				BlockTimestamp:    ts,
				BlockHeight:       bh,
			})
		}
	}
}

type cleaner interface {
	clearTransactionsOlderThan(ctx context.Context, time time.Time)
}

func startCleaningProcess(ctx context.Context, tickInterval func() time.Duration, expiration func() time.Duration, c cleaner, logger log.BasicLogger) chan struct{} {
	stopped := make(chan struct{})
	synchronization.NewPeriodicalTrigger(ctx, tickInterval(), func() {
		c.clearTransactionsOlderThan(ctx, time.Now().Add(-1*expiration()))
	}, func() {
		close(stopped)
	})

	return stopped
}
