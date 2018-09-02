package synchronization

import "time"

type Timer struct {
	timer *time.Timer
	C     <-chan time.Time
}

func NewTimer(d time.Duration) *Timer {
	timer := time.NewTimer(d)
	return &Timer{timer: timer, C: timer.C}
}

func (t *Timer) GetTimer() *time.Timer {
	return t.timer
}

func (t *Timer) Reset(d time.Duration) bool {
	active := t.Stop()
	t.timer.Reset(d)
	return active
}

func (t *Timer) Stop() bool {
	active := t.timer.Stop()
	if !active {
		<-t.C
	}
	return active
}
