package publicapi

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
	"time"
)

func newWaiter(ctx context.Context) *waiter {
	// TODO supervise
	result := &waiter{c: make(chan txWaiterMessage)}
	result.startReceiptHandler(ctx)
	return result
}

func (w *waiter) startReceiptHandler(ctx context.Context) chan struct{} {
	stopped := make(chan struct{})
	go func(ctx context.Context) {
		txChan := map[string]chan *services.AddNewTransactionOutput{}
		for {
			select {
			case message := <-w.c:
				outputChan, _ := txChan[message.txId]
				if message.c != nil && outputChan == nil { // first request
					txChan[message.txId] = message.c
				}
				// TODO - handle the case of a second request for the same transaction while someone else is already waiting
				if message.output != nil && outputChan != nil { // send output and cleanup
					select {
					case outputChan <- message.output:
					default:
					}
					close(outputChan)
					outputChan = nil
					delete(txChan, message.txId)
				}
				if message.cleanup && message.c == outputChan && outputChan != nil { // cleanup
					close(outputChan)
					outputChan = nil
					delete(txChan, message.txId)
				}

			case <-ctx.Done():
				close(stopped)
				return
			}
		}
	}(ctx)
	return stopped
}

type waiter struct {
	c chan txWaiterMessage
}

func (w *waiter) wait(txHash primitives.Sha256, timeout time.Duration) (*services.AddNewTransactionOutput, error) {
	receiptChannel := make(chan *services.AddNewTransactionOutput)

	w.c <- txWaiterMessage{
		txId: txHash.KeyForMap(),
		c:    receiptChannel,
	}
	defer func() {
		w.c <- txWaiterMessage{
			txId:    txHash.KeyForMap(),
			c:       receiptChannel,
			cleanup: true,
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	var ta *services.AddNewTransactionOutput
	select {
	case <-timer.C:
		return nil, errors.Errorf("timed out waiting for transaction result")
	case ta = <-receiptChannel:
		return ta, nil
	}
}
func (w *waiter) HandleTransactionResults(input *handlers.HandleTransactionResultsInput) (*handlers.HandleTransactionResultsOutput, error) {
	for _, txReceipt := range input.TransactionReceipts {
		select {
		case w.c <- txWaiterMessage{
			txId: txReceipt.Txhash().KeyForMap(),
			output: &services.AddNewTransactionOutput{
				TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
				TransactionReceipt: txReceipt,
				BlockHeight:        input.BlockHeight,
				BlockTimestamp:     input.Timestamp,
			},
		}:
		default:
		}
		// if we have no one to wait we just ignore this receipt ... can be accessed via getstatus
	}
	return &handlers.HandleTransactionResultsOutput{}, nil
}
