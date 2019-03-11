package transactionpool

type requestConcurrencyLimiter struct {
	slots chan struct{}
}

func NewRequestConcurrencyLimiter(maxConcurrentRequests int) *requestConcurrencyLimiter {
	return &requestConcurrencyLimiter{
		slots: make(chan struct{}, maxConcurrentRequests),
	}
}

func (r *requestConcurrencyLimiter) RequestSlot() {
	r.slots <- struct{}{}
}

func (r *requestConcurrencyLimiter) ReleaseSlot() {
	<-r.slots
}
