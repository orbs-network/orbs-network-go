package synchronization

import "time"

// This struct comes to work around the timer channel issue: https://github.com/golang/go/issues/11513
// Google couldn't break the API or behavior, so they documented it https://github.com/golang/go/issues/14383
// we just wrap the timer so we can reset and stop as expected without the workaround of the channel issue.
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
