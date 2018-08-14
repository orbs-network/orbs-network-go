package transactionpool

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"time"
)

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
	PendingPoolSizeInBytes() uint32
	TransactionExpirationWindowInSeconds() uint32
	FutureTimestampGraceInSeconds() uint32
	VirtualChainId() primitives.VirtualChainId
}

type service struct {
	gossip                     gossiptopics.TransactionRelay
	virtualMachine             services.VirtualMachine
	transactionResultsHandlers []handlers.TransactionResultsHandler
	log                        instrumentation.BasicLogger
	config                     Config

	lastCommittedBlockHeight primitives.BlockHeight
	pendingPool              *pendingTxPool
	committedPool            *committedTxPool
}

func NewTransactionPool(gossip gossiptopics.TransactionRelay, virtualMachine services.VirtualMachine, config Config, reporting instrumentation.BasicLogger) services.TransactionPool {
	s := &service{
		gossip:         gossip,
		virtualMachine: virtualMachine,
		config:         config,
		log:            reporting.For(instrumentation.Service("transaction-pool")),

		pendingPool:   NewPendingPool(config),
		committedPool: NewCommittedPool(),
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
			s.log.Error("error adding forwarded transaction to pending pool", instrumentation.Error(err), instrumentation.Stringable("transaction", tx))
		}
	}
	return nil, nil
}

func (s *service) createValidationContext() *validationContext {
	return &validationContext{
		expiryWindow:                time.Duration(s.config.TransactionExpirationWindowInSeconds()) * time.Second,
		lastCommittedBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()), //TODO use real time stamp
		futureTimestampGrace:        time.Duration(s.config.FutureTimestampGraceInSeconds()) * time.Second,
		virtualChainId:              s.config.VirtualChainId(),
	}
}
