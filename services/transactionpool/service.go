package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
	PendingPoolSizeInBytes() uint32
	TransactionExpirationWindowInSeconds() time.Duration
	FutureTimestampGraceInSeconds() uint32
	VirtualChainId() primitives.VirtualChainId
	QuerySyncGraceBlockDist() uint32
	QueryGraceTimeoutMillis() time.Duration
}

type service struct {
	gossip                     gossiptopics.TransactionRelay
	virtualMachine             services.VirtualMachine
	transactionResultsHandlers []handlers.TransactionResultsHandler
	logger                     log.BasicLogger
	config                     Config

	lastCommittedBlockHeight    primitives.BlockHeight
	lastCommittedBlockTimestamp primitives.TimestampNano
	pendingPool                 *pendingTxPool
	committedPool               *committedTxPool
	blockTracker                *synchronization.BlockTracker
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay,
	virtualMachine services.VirtualMachine,
	config Config,
	logger log.BasicLogger,
	initialTimestamp primitives.TimestampNano) services.TransactionPool {
	s := &service{
		gossip:         gossip,
		virtualMachine: virtualMachine,
		config:         config,
		logger:         logger.For(log.Service("transaction-pool")),

		lastCommittedBlockTimestamp: initialTimestamp, // this is so that we do not reject transactions on startup, before any block has been committed
		pendingPool:                 NewPendingPool(config),
		committedPool:               NewCommittedPool(),
		blockTracker:                synchronization.NewBlockTracker(0, uint16(config.QuerySyncGraceBlockDist()), time.Duration(config.QueryGraceTimeoutMillis())),
	}
	gossip.RegisterTransactionRelayHandler(s)
	return s
}

func (s *service) GetCommittedTransactionReceipt(input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {
	panic("Not implemented")
}

func (s *service) ValidateTransactionsForOrdering(input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	panic("Not implemented")
}

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	s.transactionResultsHandlers = append(s.transactionResultsHandlers, handler)
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {

	//TODO verify message signature
	for _, tx := range input.Message.SignedTransactions {
		if _, err := s.pendingPool.add(tx, input.Message.Sender.SenderPublicKey()); err != nil {
			s.logger.Error("error adding forwarded transaction to pending pool", log.Error(err), log.Stringable("transaction", tx))
		}
	}
	return nil, nil
}

func (s *service) createValidationContext() *validationContext {
	return &validationContext{
		expiryWindow:                s.config.TransactionExpirationWindowInSeconds(),
		lastCommittedBlockTimestamp: s.lastCommittedBlockTimestamp,
		futureTimestampGrace:        time.Duration(s.config.FutureTimestampGraceInSeconds()) * time.Second,
		virtualChainId:              s.config.VirtualChainId(),
	}
}
