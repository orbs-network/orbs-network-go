package transactionpool

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"time"
)

func (s *service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	s.transactionResultsHandlers = append(s.transactionResultsHandlers, handler)
}

func (s *service) HandleForwardedTransactions(input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	sender := input.Message.Sender
	oneBigHash, _ := HashTransactions(input.Message.SignedTransactions)

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

func (s *service) startForwardingProcess(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Millisecond):
				s.drainQueueAndForwardTransactions()
			}
		}

	}()
}

func (s *service) drainQueueAndForwardTransactions() {
	txs := s.drainQueue()
	if len(txs) == 0 {
		return
	}

	oneBigHash, hashes := HashTransactions(txs)

	sig, err := signature.SignEd25519(s.config.NodePrivateKey(), oneBigHash)
	if err != nil {
		s.logger.Error("error signing transactions", log.Error(err), log.StringableSlice("transactions", txs))
		return
	}

	_, err = s.gossip.BroadcastForwardedTransactions(&gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: txs,
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
				Signature:       sig,
			}).Build(),
		},
	})

	for _, hash := range hashes {
		if err != nil {
			s.logger.Info("failed forwarding transaction via gossip", log.Error(err), log.String("flow", "checkpoint"), log.Stringable("txHash", hash))
		} else {
			s.logger.Info("forwarded transaction via gossip", log.String("flow", "checkpoint"), log.Stringable("txHash", hash))
		}
	}
}

func (s *service) drainQueue() []*protocol.SignedTransaction {
	s.forwardQueueMutex.Lock()
	txs := s.forwardQueue
	s.forwardQueue = nil
	s.forwardQueueMutex.Unlock()
	return txs
}

func HashTransactions(txs []*protocol.SignedTransaction) (oneBigHash []byte, hashes []primitives.Sha256) {
	for _, tx := range txs {
		hash := digest.CalcTxHash(tx.Transaction())

		hashes = append(hashes, hash)
		oneBigHash = append(oneBigHash, hash...)
	}

	return
}
