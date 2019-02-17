package transactionpool

import "context"

//Kind of a barrier which waits until a specific number of notifications have been met, or until a context is done
//Note: not thread-safe; do not reuse the same instance in two goroutines
type transactionWaiter struct {
	incremented chan struct{}
}

func (w *transactionWaiter) waitForIncomingTransaction(ctx context.Context) bool {
	for {
		select {
		case <-w.incremented:
			return true
		case <-ctx.Done():
			return false
		}
	}
}

func (w *transactionWaiter) inc(ctx context.Context) {
	select {
	case w.incremented <- struct{}{}:
	default:
		return
	}
}

func newTransactionWaiter() *transactionWaiter {
	return &transactionWaiter{incremented: make(chan struct{}, 1)}
}
