package transactionpool

type requestRateLimiter struct {
	slots chan struct{}
}

func NewRequestRateLimiter(maxConcurrentRequests int) *requestRateLimiter {
	return &requestRateLimiter{
		slots: make(chan struct{}, maxConcurrentRequests),
	}
}

func (r *requestRateLimiter) RequestSlot() {
	r.slots <- struct{}{}
}

func (r *requestRateLimiter) ReleaseSlot() {
	<-r.slots
}
