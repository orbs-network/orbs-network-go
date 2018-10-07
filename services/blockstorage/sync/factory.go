package sync

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
)

type stateFactory struct {
	config  blockSyncConfig
	gossip  gossiptopics.BlockSync
	storage BlockSyncStorage
	logger  log.BasicLogger
}

func NewStateFactory(config blockSyncConfig, gossip gossiptopics.BlockSync, storage BlockSyncStorage, logger log.BasicLogger) *stateFactory {
	return &stateFactory{
		config:  config,
		gossip:  gossip,
		storage: storage,
		logger:  logger,
	}
}

func (f *stateFactory) CreateIdleState() syncState {
	return &idleState{
		sf:            f,
		config:        f.config,
		noCommitTimer: synchronization.NewTimer(f.config.BlockSyncNoCommitInterval()),
		restartIdle:   make(chan struct{}),
	}
}

func (f *stateFactory) CreateCollectingAvailabilityResponseState() syncState {
	return &collectingAvailabilityResponsesState{
		sf:      f,
		gossip:  f.gossip,
		storage: f.storage,
		config:  f.config,
		logger:  f.logger,
	}
}

func (f *stateFactory) CreateFinishedCARState() syncState {
	return &finishedCARState{}
}
