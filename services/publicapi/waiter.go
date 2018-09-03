package publicapi

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"time"
)

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
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

const retryCount = 10
const retryDelay = 10 * time.Millisecond

type txWaiterMessage struct {
	txId       string
	c          txResultChan
	output     *services.AddNewTransactionOutput
	cleanup    bool
	retryCount byte
}

func newTxWaiter(ctx context.Context) *txWaiter {
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
						if message.retryCount > 0 {
							message.retryCount--
							go func() {
								time.Sleep(retryDelay)
								w.tryEnqueue(&message)
							}()
						}
					}
				}
				if message.cleanup && message.c == outputChan && outputChan != nil { // cleanup
					close(outputChan)
					outputChan = nil
					delete(txChan, message.txId)
				}

			case <-ctx.Done():
				close(w.queue)
				for _, c := range txChan {
					close(c)
				}
				close(w.stopped)
				return
			}
		}
	}(ctx)
}

func (w *txWaiter) forget(txHash primitives.Sha256, c txResultChan) {
	w.tryEnqueue(&txWaiterMessage{
		txId:    txHash.KeyForMap(),
		c:       c,
		cleanup: true,
	})
}

func (w *txWaiter) reportCompleted(receipt *protocol.TransactionReceipt, blockHeight primitives.BlockHeight, timestampNano primitives.TimestampNano) {
	w.tryEnqueue(&txWaiterMessage{
		txId:       receipt.Txhash().KeyForMap(),
		retryCount: retryCount,
		output: &services.AddNewTransactionOutput{
			TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
			TransactionReceipt: receipt,
			BlockHeight:        blockHeight,
			BlockTimestamp:     timestampNano,
		},
	})
}

func (w *txWaiter) createTxWaitCtx(txHash primitives.Sha256) (waitContext *txWaitContext) {
	receiptChannel := make(txResultChan)
	waitContext = &txWaitContext{c: receiptChannel, txHash: txHash, waiter: w}

	defer func() {
		if p := recover(); p != nil {
			close(waitContext.c)
		}
	}()
	w.queue <- txWaiterMessage{
		txId: txHash.KeyForMap(),
		c:    receiptChannel,
	}

	return
}

func (w *txWaiter) tryEnqueue(message *txWaiterMessage) {
	defer func() { recover() }()
	w.queue <- *message
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
