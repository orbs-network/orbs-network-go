package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
)

var LogTag = log.Service("transaction-pool")

type BlockHeightReporter interface {
	IncrementTo(height primitives.BlockHeight)
}

type service struct {
	gossip                     gossiptopics.TransactionRelay
	virtualMachine             services.VirtualMachine
	blockHeightReporter        BlockHeightReporter // used to allow test to wait for a block height to reach the transaction pool
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
	transactionWaiter    *transactionWaiter

	metrics struct {
		blockHeight *metric.Gauge
	}
}

func (s *service) lastCommittedBlockHeightAndTime() (primitives.BlockHeight, primitives.TimestampNano) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.lastCommittedBlockHeight, s.mu.lastCommittedBlockTimestamp
}

func (s *service) createValidationContext() *validationContext {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return &validationContext{
		expiryWindow:                s.config.TransactionExpirationWindow(),
		lastCommittedBlockTimestamp: s.mu.lastCommittedBlockTimestamp,
		futureTimestampGrace:        s.config.TransactionPoolFutureTimestampGraceTimeout(),
		virtualChainId:              s.config.VirtualChainId(),
	}
}
