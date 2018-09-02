package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"time"
)

type txWaiter struct {
	queue   chan txWaiterMessage
	stopped chan struct{}
}

type txWaitContext struct {
	c      txResultChan
	txHash primitives.Sha256
	waiter *txWaiter
}

type txResultChan chan *services.AddNewTransactionOutput

type txWaiterMessage struct {
	txId    string
	c       txResultChan
	output  *services.AddNewTransactionOutput
	cleanup bool
}

func newWaiter(ctx context.Context) *txWaiter {
	// TODO supervise
	result := &txWaiter{queue: make(chan txWaiterMessage)}
	result.startReceiptHandler(ctx)
	return result
}

func (w *txWaiter) startReceiptHandler(ctx context.Context) {
	w.stopped = make(chan struct{})
	go func(ctx context.Context) {
		txChan := map[string]txResultChan{}
		for {
			select {
			case message := <-w.queue:
				outputChan, _ := txChan[message.txId]
				if message.c != nil && outputChan == nil && !message.cleanup { // first request
					txChan[message.txId] = message.c
				}
				if message.c != nil && outputChan != nil && !message.cleanup { // second request
					close(outputChan)
					outputChan = nil
					txChan[message.txId] = message.c
				}
				if message.output != nil && outputChan != nil && !message.cleanup { // send output and cleanup
					select {
					case outputChan <- message.output:
						close(outputChan)
						outputChan = nil
						delete(txChan, message.txId)
					default:
						go func() {
							w.queue <- message
						}()
					}
				}
				if message.cleanup && message.c == outputChan && outputChan != nil { // cleanup
					close(outputChan)
					outputChan = nil
					delete(txChan, message.txId)
				}

			case <-ctx.Done():
				close(w.stopped)
				return
			}
		}
	}(ctx)
}

func (w *txWaiter) startWaiting(txHash primitives.Sha256, c txResultChan) {
	w.queue <- txWaiterMessage{
		txId: txHash.KeyForMap(),
		c:    c,
	}
}

func (w *txWaiter) forget(txHash primitives.Sha256, c txResultChan) {
	w.queue <- txWaiterMessage{
		txId:    txHash.KeyForMap(),
		c:       c,
		cleanup: true,
	}
}

func (w *txWaiter) reportCompleted(receipt *protocol.TransactionReceipt, blockHeight primitives.BlockHeight, timestampNano primitives.TimestampNano) {
	w.queue <- txWaiterMessage{
		txId: receipt.Txhash().KeyForMap(),
		output: &services.AddNewTransactionOutput{
			TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
			TransactionReceipt: receipt,
			BlockHeight:        blockHeight,
			BlockTimestamp:     timestampNano,
		},
	}
}

func (w *txWaiter) prepareFor(txHash primitives.Sha256) *txWaitContext {
	receiptChannel := make(txResultChan)

	w.startWaiting(txHash, receiptChannel)

	return &txWaitContext{c: receiptChannel, txHash: txHash, waiter: w}
}

func (w *txWaitContext) until(timeout time.Duration) (*services.AddNewTransactionOutput, error) {

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, errors.Errorf("timed out waiting for transaction result")
	case ta, open := <-w.c:
		if !open {
			return nil, errors.Errorf("waiting aborted")
		}
		return ta, nil
	}
}

func (w *txWaitContext) cleanup() {
	w.waiter.forget(w.txHash, w.c)
}
