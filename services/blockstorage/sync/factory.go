package sync

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
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
		sf:          f,
		config:      f.config,
		logger:      f.logger,
		restartIdle: make(chan struct{}),
	}
}

func (f *stateFactory) CreateCollectingAvailabilityResponseState() syncState {
	return &collectingAvailabilityResponsesState{
		sf:           f,
		gossipClient: newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncBatchSize(), f.config.NodePublicKey()),
		config:       f.config,
		logger:       f.logger,
		responsesC:   make(chan *gossipmessages.BlockAvailabilityResponseMessage),
	}
}

func (f *stateFactory) CreateFinishedCARState(responses []*gossipmessages.BlockAvailabilityResponseMessage) syncState {
	return &finishedCARState{
		responses: responses,
		logger:    f.logger,
		sf:        f,
	}
}

func (f *stateFactory) CreateWaitingForChunksState(sourceKey primitives.Ed25519PublicKey) syncState {
	return &waitingForChunksState{
		sourceKey:    sourceKey,
		sf:           f,
		gossipClient: newBlockSyncGossipClient(f.gossip, f.storage, f.logger, f.config.BlockSyncBatchSize(), f.config.NodePublicKey()),
		config:       f.config,
		logger:       f.logger,
		blocksC:      make(chan *gossipmessages.BlockSyncResponseMessage),
		abort:        make(chan struct{}),
	}
}

func (f *stateFactory) CreateProcessingBlocksState(message *gossipmessages.BlockSyncResponseMessage) syncState {
	return &processingBlocksState{
		blocks:  message,
		sf:      f,
		logger:  f.logger,
		storage: f.storage,
	}
}
