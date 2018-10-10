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
	"time"
)

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

	oneBigHash, hashes := hashTransactions(txs)

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
			s.logger.Info("failed forwarding transaction via gossip", log.Error(err), log.Stringable("txHash", hash))
		} else {
			s.logger.Info("forwarded transaction via gossip", log.Stringable("txHash", hash))
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

func hashTransactions(txs []*protocol.SignedTransaction) (oneBigHash []byte, hashes []primitives.Sha256) {
	for _, tx := range txs {
		hash := digest.CalcTxHash(tx.Transaction())

		hashes = append(hashes, hash)
		oneBigHash = append(oneBigHash, hash...)
	}

	return
}
