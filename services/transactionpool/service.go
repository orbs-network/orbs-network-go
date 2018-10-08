package transactionpool

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"time"
)

var LogTag = log.Service("transaction-pool")

type Config interface {
	NodePublicKey() primitives.Ed25519PublicKey
	NodePrivateKey() primitives.Ed25519PrivateKey
	VirtualChainId() primitives.VirtualChainId
	BlockTrackerGraceDistance() uint32
	BlockTrackerGraceTimeout() time.Duration
	TransactionPoolPendingPoolSizeInBytes() uint32
	TransactionPoolTransactionExpirationWindow() time.Duration
	TransactionPoolFutureTimestampGraceTimeout() time.Duration
	TransactionPoolPendingPoolClearExpiredInterval() time.Duration
	TransactionPoolCommittedPoolClearExpiredInterval() time.Duration
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

func NewTransactionPool(ctx context.Context,
	gossip gossiptopics.TransactionRelay,
	virtualMachine services.VirtualMachine,
	config Config,
	logger log.BasicLogger) services.TransactionPool {
	pendingPool := NewPendingPool(config.TransactionPoolPendingPoolSizeInBytes)

	s := &service{
		gossip:         gossip,
		virtualMachine: virtualMachine,
		config:         config,
		logger:         logger.WithTags(LogTag),

		lastCommittedBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()), // this is so that we do not reject transactions on startup, before any block has been committed
		pendingPool:                 pendingPool,
		committedPool:               NewCommittedPool(),
		blockTracker:                synchronization.NewBlockTracker(0, uint16(config.BlockTrackerGraceDistance()), time.Duration(config.BlockTrackerGraceTimeout())),
	}

	gossip.RegisterTransactionRelayHandler(s)
	pendingPool.onTransactionRemoved = s.onTransactionError

	//TODO supervise
	startCleaningProcess(ctx, config.TransactionPoolCommittedPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.committedPool, logger)
	startCleaningProcess(ctx, config.TransactionPoolPendingPoolClearExpiredInterval, config.TransactionPoolTransactionExpirationWindow, s.pendingPool, logger)

	return s
}

func (s *service) GetCommittedTransactionReceipt(input *services.GetCommittedTransactionReceiptInput) (*services.GetCommittedTransactionReceiptOutput, error) {

	tsWithGrace := s.lastCommittedBlockTimestamp + primitives.TimestampNano(s.config.TransactionPoolFutureTimestampGraceTimeout().Nanoseconds())
	if input.TransactionTimestamp > tsWithGrace {
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

func (s *service) ValidateTransactionsForOrdering(input *services.ValidateTransactionsForOrderingInput) (*services.ValidateTransactionsForOrderingOutput, error) {
	if err := s.blockTracker.WaitForBlock(input.BlockHeight); err != nil {
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
	preOrderResults, _ := s.virtualMachine.TransactionSetPreOrder(&services.TransactionSetPreOrderInput{
		SignedTransactions: input.SignedTransactions,
		BlockHeight:        s.lastCommittedBlockHeight,
	})

	for i, tx := range input.SignedTransactions {
		if status := preOrderResults.PreOrderResults[i]; status != protocol.TRANSACTION_STATUS_PRE_ORDER_VALID {
			return nil, errors.Errorf("transaction with hash %s failed pre-order checks with status %s", digest.CalcTxHash(tx.Transaction()), status)
		}
	}
	return &services.ValidateTransactionsForOrderingOutput{}, nil
}

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	s.transactionResultsHandlers = append(s.transactionResultsHandlers, handler)
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {

	//TODO this is copying and needs to go away pending issue #119
	sender := input.Message.Sender
	var allTransactions []byte
	for _, tx := range input.Message.SignedTransactions {
		allTransactions = append(allTransactions, tx.Raw()...)
	}

	if !signature.VerifyEd25519(sender.SenderPublicKey(), allTransactions, sender.Signature()) {
		return nil, errors.Errorf("invalid signature in relay message from sender %s", sender.SenderPublicKey())
	}

	for _, tx := range input.Message.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		s.logger.Info("adding forwarded transaction to the pool", log.String("flow", "checkpoint"), log.Stringable("transaction", tx), log.Stringable("txHash", txHash))
		if _, err := s.pendingPool.add(tx, sender.SenderPublicKey()); err != nil {
			s.logger.Error("error adding forwarded transaction to pending pool", log.Error(err), log.Stringable("transaction", tx), log.Stringable("txHash", txHash))
		}
	}
	return nil, nil
}

func (s *service) createValidationContext() *validationContext {
	if s.lastCommittedBlockTimestamp == 0 {
		panic("last committed block timestamp should never be zero!")
	}
	return &validationContext{
		expiryWindow:                s.config.TransactionPoolTransactionExpirationWindow(),
		lastCommittedBlockTimestamp: s.lastCommittedBlockTimestamp,
		futureTimestampGrace:        s.config.TransactionPoolFutureTimestampGraceTimeout(),
		virtualChainId:              s.config.VirtualChainId(),
	}
}

func (s *service) getTxResult(receipt *protocol.TransactionReceipt, status protocol.TransactionStatus) *services.GetCommittedTransactionReceiptOutput {
	return &services.GetCommittedTransactionReceiptOutput{
		TransactionStatus:  status,
		TransactionReceipt: receipt,
		BlockHeight:        s.lastCommittedBlockHeight,
		BlockTimestamp:     s.lastCommittedBlockTimestamp,
	}
}

func (s *service) onTransactionError(txHash primitives.Sha256, removalReason protocol.TransactionStatus) {
	if removalReason != protocol.TRANSACTION_STATUS_COMMITTED {
		for _, trh := range s.transactionResultsHandlers {
			trh.HandleTransactionError(&handlers.HandleTransactionErrorInput{
				Txhash:            txHash,
				TransactionStatus: removalReason,
				BlockTimestamp:    s.lastCommittedBlockTimestamp,
				BlockHeight:       s.lastCommittedBlockHeight,
			})
		}
	}
}

type cleaner interface {
	clearTransactionsOlderThan(time time.Time)
}

// TODO supervise
func startCleaningProcess(ctx context.Context, tickInterval func() time.Duration, expiration func() time.Duration, c cleaner, logger log.BasicLogger) chan struct{} {
	stopped := make(chan struct{})
	ticker := time.NewTicker(tickInterval())
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// TODO: in production we need to restart our long running goroutine (decide on supervision mechanism)
				logger.Error("panic in TransactionPool.cleaningProcess long running goroutine", log.String("panic", fmt.Sprintf("%v", r)))
			}
		}()

		for {
			select {
			case <-ctx.Done():
				close(stopped)
				return
			case <-ticker.C:
				c.clearTransactionsOlderThan(time.Now().Add(-1 * expiration()))
			}
		}

	}()
	return stopped
}
