package sync

import (
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"time"
)

type syncState interface {
	name() string
	next() syncState
	blockCommitted(blockHeight primitives.BlockHeight)
	gotAvailabilityResponse(message gossipmessages.BlockAvailabilityResponseMessage)
	gotBlocks(source primitives.Ed25519PublicKey, blocks []*protocol.BlockPairContainer)
}

type blockSync struct {
	reporting        log.BasicLogger
	lastBlockHeight  primitives.BlockHeight
	idleStateTimeout time.Duration
	shouldStop       bool
}

func NewBlockSync(bh primitives.BlockHeight, idleStateTimeout time.Duration) *blockSync {
	bs := &blockSync{
		reporting:        log.GetLogger(log.Source("block-sync")),
		lastBlockHeight:  bh,
		idleStateTimeout: idleStateTimeout,
		shouldStop:       false,
	}

	go bs.syncLoop()
	return bs
}

func (bs *blockSync) Shutdown() {
	bs.shouldStop = true
}

func (bs *blockSync) syncLoop() {
	bs.reporting.Info("starting block sync main loop")
	var state syncState = nil
	for state = createIdleState(bs.idleStateTimeout); state != nil && !bs.shouldStop; {
		bs.reporting.Info("state transitioning", log.String("current-state", state.name()))
		state = state.next()
	}

	bs.reporting.Info("block sync main loop ended")
}
