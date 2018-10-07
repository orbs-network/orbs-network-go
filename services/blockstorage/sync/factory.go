package sync

import (
	"github.com/orbs-network/orbs-network-go/synchronization"
	"time"
)

type stateFactory struct{}

func NewStateFactory() *stateFactory {
	return &stateFactory{}
}

func (f *stateFactory) CreateIdleState(d time.Duration) syncState {
	return &idleState{
		sf:              f,
		noCommitTimeout: d,
		noCommitTimer:   synchronization.NewTimer(d),
		restartIdle:     make(chan struct{}),
	}
}
