// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package synchronization

import "time"

// This struct comes to work around the timer channel issue: https://github.com/golang/go/issues/11513
// Google couldn't break the API or behavior, so they documented it https://github.com/golang/go/issues/14383
// we just wrap the timer so we can reset and stop as expected without the workaround of the channel issue.
type Timer struct {
	timer *time.Timer
	C     <-chan time.Time

	writableC *chan time.Time // the same as C just writable and not exported, used by NewTimerWithManualTick()
}

func NewTimer(d time.Duration) *Timer {
	timer := time.NewTimer(d)
	return &Timer{timer: timer, C: timer.C}
}

func (t *Timer) GetTimer() *time.Timer {
	return t.timer
}

func (t *Timer) Reset(d time.Duration) bool {
	if t.timer == nil {
		return false
	}

	active := t.Stop()
	t.timer.Reset(d)
	return active
}

func (t *Timer) Stop() bool {
	if t.timer == nil {
		return false
	}

	active := t.timer.Stop()
	if !active {
		select {
		case <-t.C:
		default:
		}
	}
	return active
}

// used primarily for tests
func (t *Timer) ManualTick() {
	if t.writableC != nil {
		go func() { // ManualTick is expected to be non blocking
			*t.writableC <- time.Now()
		}()
	}
}

// used primarily for tests
func NewTimerWithManualTick() *Timer {
	c := make(chan time.Time)
	return &Timer{
		C:         c,
		writableC: &c,
	}
}
