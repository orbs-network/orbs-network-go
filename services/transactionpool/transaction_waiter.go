package transactionpool

import "context"

//Kind of a barrier which waits until a specific number of notifications have been met, or until a context is done
//Note: not thread-safe; do not reuse the same instance in two goroutines
type transactionWaiter struct {
	incremented chan struct{}
	waiting     bool
}

func (w *transactionWaiter) waitFor(ctx context.Context, numOfNotificationsToWaitFor int) chan bool {
	ch := make(chan bool)
	w.waiting = true
	notificationsMet := 0
	go func() {
		for {
			select {
			case <-w.incremented:
				notificationsMet++
				if notificationsMet >= numOfNotificationsToWaitFor {
					ch <- true
					w.waiting = false
					return
				}
			case <-ctx.Done():
				ch <- false
				w.waiting = false
				return
			}
		}
	}()
	return ch
}

func (w *transactionWaiter) inc(ctx context.Context) {
	if !w.waiting {
		return
	}
	go func() { // so that we don't block anyone incrementing
		select {
		case w.incremented <- struct{}{}:
		case <-ctx.Done():
		}
	}()
}

func newTransactionWaiter() *transactionWaiter {
	return &transactionWaiter{incremented: make(chan struct{})}
}
