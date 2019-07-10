package adapter

import "time"

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
	return time.Now().Add(c.delta)
}

func (c *AdjustableClock) AddSeconds(delta int) {
	c.delta += time.Duration(delta) * time.Second
}

func NewAdjustableClock() *AdjustableClock {
	return &AdjustableClock{}
}
