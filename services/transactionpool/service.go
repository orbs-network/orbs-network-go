package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"sync"
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

func (s *service) currentBlockHeightAndTime() (primitives.BlockHeight, primitives.TimestampNano) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mu.lastCommittedBlockHeight, s.mu.lastCommittedBlockTimestamp
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
