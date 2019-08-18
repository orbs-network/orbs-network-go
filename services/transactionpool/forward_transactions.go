// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/crypto/signer"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"hash/adler32"
	"sync"
	"time"
)

type TransactionForwarderConfig interface {
	NodeAddress() primitives.NodeAddress
	TransactionPoolPropagationBatchSize() uint16
	TransactionPoolPropagationBatchingTimeout() time.Duration
}

func (s *Service) RegisterTransactionResultsHandler(handler handlers.TransactionResultsHandler) {
	s.transactionResultsHandlers.Lock()
	defer s.transactionResultsHandlers.Unlock()
	s.transactionResultsHandlers.handlers = append(s.transactionResultsHandlers.handlers, handler)
}

func (s *Service) HandleForwardedTransactions(ctx context.Context, input *gossiptopics.ForwardedTransactionsInput) (*gossiptopics.EmptyOutput, error) {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	sender := input.Message.Sender
	oneBigHash, _, err := HashTransactions(input.Message.SignedTransactions...)
	if err != nil {
		return nil, errors.Wrapf(err, "could not create one hash, invalid signature in relay message from sender %s", sender.SenderNodeAddress())
	}

	if err := digest.VerifyNodeSignature(sender.SenderNodeAddress(), oneBigHash, sender.Signature()); err != nil {
		return nil, errors.Wrapf(err, "invalid signature in relay message from sender %s", sender.SenderNodeAddress())
	}

	for _, tx := range input.Message.SignedTransactions {
		txHash := digest.CalcTxHash(tx.Transaction())
		logger.Info("adding forwarded transaction to the pool", log.String("flow", "checkpoint"), logfields.Transaction(txHash))
		if _, err := s.pendingPool.add(tx, sender.SenderNodeAddress()); err != nil {
			logger.Error("error adding forwarded transaction to pending pool", log.Error(err), log.Stringable("transaction", tx), logfields.Transaction(txHash))
		}
	}
	return nil, nil
}

type transactionForwarder struct {
	govnr.TreeSupervisor
	logger log.Logger
	config TransactionForwarderConfig
	gossip gossiptopics.TransactionRelay
	signer signer.Signer

	forwardQueueMutex *sync.Mutex
	forwardQueue      []*protocol.SignedTransaction
	transactionAdded  chan uint16
}

func NewTransactionForwarder(ctx context.Context, logger log.Logger, signer signer.Signer, config TransactionForwarderConfig, gossip gossiptopics.TransactionRelay) *transactionForwarder {
	f := &transactionForwarder{
		logger:            logger.WithTags(log.String("component", "transaction-forwarder")),
		config:            config,
		gossip:            gossip,
		signer:            signer,
		forwardQueueMutex: &sync.Mutex{},
		transactionAdded:  make(chan uint16, 1), // buffered channel because we don't always have a receiver to read
	}

	f.start(ctx)

	return f
}

func (f *transactionForwarder) submit(transactions ...*protocol.SignedTransaction) {
	f.transactionAdded <- f.appendToQueue(transactions)
}

func (f *transactionForwarder) appendToQueue(transactions []*protocol.SignedTransaction) uint16 {
	f.forwardQueueMutex.Lock()
	defer f.forwardQueueMutex.Unlock()
	f.forwardQueue = append(f.forwardQueue, transactions...)
	return uint16(len(f.forwardQueue))
}

func (f *transactionForwarder) start(parent context.Context) {
	f.Supervise(govnr.Forever(parent, "transaction forwarder", logfields.GovnrErrorer(f.logger), func() {
		f.logger.Info("transaction forwarder started")
		for {
			ctx := trace.NewContext(parent, "TransactionForwarder")
			timer := synchronization.NewTimer(f.config.TransactionPoolPropagationBatchingTimeout())

			select {
			case <-ctx.Done():
				return
			case txCount := <-f.transactionAdded:
				if txCount >= f.config.TransactionPoolPropagationBatchSize() {
					timer.Stop()
					f.drainQueueAndForward(ctx)
				}
			case <-timer.C:
				f.drainQueueAndForward(ctx)
			}
		}
	}))
}

func (f *transactionForwarder) drainQueueAndForward(ctx context.Context) {
	logger := f.logger.WithTags(trace.LogFieldFrom(ctx))
	txs := f.drainQueue()
	if len(txs) == 0 {
		return
	}

	oneBigHash, hashes, err := HashTransactions(txs...)
	if err != nil {
		logger.Error("error creating one big hash while signing transactions", log.Error(err), log.StringableSlice("transactions", txs))
		f.submit(txs...)
		return
	}

	sig, err := f.signer.Sign(ctx, oneBigHash)
	if err != nil {
		logger.Error("error signing transactions", log.Error(err), log.StringableSlice("transactions", txs))
		f.submit(txs...)
		return
	}

	_, err = f.gossip.BroadcastForwardedTransactions(ctx, &gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			SignedTransactions: txs,
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: f.config.NodeAddress(),
				Signature:         sig,
			}).Build(),
		},
	})

	for _, hash := range hashes {
		if err != nil {
			logger.Info("failed forwarding transaction via gossip", log.Error(err), log.String("flow", "checkpoint"), logfields.Transaction(hash))
		} else {
			logger.Info("forwarded transaction via gossip", log.String("flow", "checkpoint"), logfields.Transaction(hash))
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

func HashTransactions(txs ...*protocol.SignedTransaction) (oneBigHash []byte, hashes []primitives.Sha256, err error) {
	checksum := adler32.New() // TODO(https://github.com/orbs-network/orbs-spec/issues/134): this needs to update to a bigger checksum/hash
	for _, tx := range txs {
		hash := digest.CalcTxHash(tx.Transaction())

		hashes = append(hashes, hash)
		_, err = checksum.Write(hash)
		if err != nil {
			return nil, nil, err
		}
	}

	oneBigHash = checksum.Sum(nil)
	return
}
