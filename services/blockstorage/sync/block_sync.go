package sync

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type syncState interface {
	name() string
	processState(ctx context.Context) syncState
	blockCommitted(blockHeight primitives.BlockHeight)
	gotAvailabilityResponse(message gossipmessages.BlockAvailabilityResponseMessage)
	gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer)
}

type blockSync struct {
	logger           log.BasicLogger
	lastBlockHeight  primitives.BlockHeight
	idleStateTimeout time.Duration
	shouldStop       bool
	sf               *stateFactory
}

func NewBlockSync(ctx context.Context, bh primitives.BlockHeight, idleStateTimeout time.Duration) *blockSync {
	bs := &blockSync{
		logger:           log.GetLogger(log.Source("block-sync")),
		lastBlockHeight:  bh,
		idleStateTimeout: idleStateTimeout,
		shouldStop:       false,
		sf:               NewStateFactory(),
	}

	go bs.syncLoop(ctx)
	return bs
}

func (bs *blockSync) Shutdown() {
	bs.shouldStop = true
}

func (bs *blockSync) syncLoop(ctx context.Context) {
	bs.logger.Info("starting block sync main loop")
	for state := bs.sf.CreateIdleState(bs.idleStateTimeout); state != nil && !bs.shouldStop; {
		bs.logger.Info("state transitioning", log.String("current-state", state.name()))
		state = state.processState(ctx)
	}

	bs.logger.Info("block sync main loop ended")
}
