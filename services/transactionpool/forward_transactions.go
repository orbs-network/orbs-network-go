package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"sync"
	"time"
)

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	s.transactionResultsHandlers = append(s.transactionResultsHandlers, handler)
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	sender := input.Message.Sender
	oneBigHash, _ := HashTransactions(input.Message.SignedTransactions...)

	if !signature.VerifyEd25519(sender.SenderPublicKey(), oneBigHash, sender.Signature()) {
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

type transactionForwarder struct {
	logger log.BasicLogger
	config config.TransactionForwarderConfig
	gossip gossiptopics.TransactionRelay

	forwardQueueMutex *sync.Mutex
	forwardQueue      []*protocol.SignedTransaction
}

func NewTransactionForwarder(ctx context.Context, logger log.BasicLogger, config config.TransactionForwarderConfig, gossip gossiptopics.TransactionRelay) *transactionForwarder {
	f := &transactionForwarder{
		logger:            logger.WithTags(log.String("component", "transaction-forwarder")),
		config:            config,
		gossip:            gossip,
		forwardQueueMutex: &sync.Mutex{},
	}

	f.startForwardingProcess(ctx)

	return f
}

func (f *transactionForwarder) enqueueTransactionToBeForwarded(transaction *protocol.SignedTransaction) {
	f.forwardQueueMutex.Lock()
	defer f.forwardQueueMutex.Unlock()
	f.forwardQueue = append(f.forwardQueue, transaction)
}

func (f *transactionForwarder) startForwardingProcess(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Millisecond):
				f.drainQueueAndForwardTransactions()
			}
		}

	}()
}

func (f *transactionForwarder) drainQueueAndForwardTransactions() {
	txs := f.drainQueue()
	if len(txs) == 0 {
		return
	}

	oneBigHash, hashes := HashTransactions(txs...)

	sig, err := signature.SignEd25519(f.config.NodePrivateKey(), oneBigHash)
	if err != nil {
		f.logger.Error("error signing transactions", log.Error(err), log.StringableSlice("transactions", txs))
		return
	}

	_, err = f.gossip.BroadcastForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: txs,
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: f.config.NodePublicKey(),
				Signature:       sig,
			}).Build(),
		},
	})

	for _, hash := range hashes {
		if err != nil {
			f.logger.Info("failed forwarding transaction via gossip", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", hash))
		} else {
			f.logger.Info("forwarded transaction via gossip", log.String("flow", "checkpoint"), log.Stringable("txHash", hash))
		}
	}
}

func (f *transactionForwarder) drainQueue() []*protocol.SignedTransaction {
	f.forwardQueueMutex.Lock()
	txs := f.forwardQueue
	f.forwardQueue = nil
	f.forwardQueueMutex.Unlock()
	return txs
}

func HashTransactions(txs ...*protocol.SignedTransaction) (oneBigHash []byte, hashes []primitives.Sha256) {
	for _, tx := range txs {
		hash := digest.CalcTxHash(tx.Transaction())

		hashes = append(hashes, hash)
		oneBigHash = append(oneBigHash, hash...)
	}

	return
}
