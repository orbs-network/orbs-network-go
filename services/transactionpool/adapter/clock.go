package adapter

import (
	"sync/atomic"
	"time"
)

type Clock interface {
	CurrentTime() time.Time
}

type systemClock struct{}

func NewSystemClock() *systemClock {
	return &systemClock{}
}

func (*systemClock) CurrentTime() time.Time {
	return time.Now()
}

type AdjustableClock struct {
	delta time.Duration
}

func (c *AdjustableClock) CurrentTime() time.Time {
	return time.Now().Add(c.loadDelta())
}

func (c *AdjustableClock) AddSeconds(delta int) {
	deltaPtr := (*int64)(&(c.delta))
	atomic.AddInt64(deltaPtr, int64(time.Duration(delta)*time.Second))
}

func NewAdjustableClock() *AdjustableClock {
	return &AdjustableClock{}
}

func (c *AdjustableClock) loadDelta() time.Duration {
	return time.Duration(atomic.LoadInt64((*int64)(&(c.delta))))
}
