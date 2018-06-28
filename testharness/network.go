package testharness

/*
Objects here are only for testing purposes, not to be used in real code
 */

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/gossip"
)

//type TestNetwork interface {
//}
//
type TestNetwork struct {
	Leader      bootstrap.Node
	Validator   bootstrap.Node
	LeaderLatch events.Latch
	LeaderBp    blockstorage.InMemoryBlockPersistence
	ValidatorBp blockstorage.InMemoryBlockPersistence
	Gossip      gossip.PausableGossip

	log []events.BufferedLog
}

func CreateTestNetwork() TestNetwork {
	leaderLog := events.NewBufferedLog("leader")
	leaderLatch := events.NewLatch()
	validatorLog := events.NewBufferedLog("validator")

	inMemoryGossip := gossip.NewPausableGossip()
	leaderBp := blockstorage.NewInMemoryBlockPersistence("leaderBp")
	validatorBp := blockstorage.NewInMemoryBlockPersistence("validatorBp")

	leader := bootstrap.NewNode(inMemoryGossip, leaderBp, events.NewCompositeEvents([]events.Events{leaderLog, leaderLatch}), true)
	validator := bootstrap.NewNode(inMemoryGossip, validatorBp, validatorLog, false)

	return TestNetwork{
		Leader:      leader,
		Validator:   validator,
		LeaderLatch: leaderLatch,
		LeaderBp:    leaderBp,
		ValidatorBp: validatorBp,
		Gossip:      inMemoryGossip,

		log: []events.BufferedLog{leaderLog, validatorLog},
	}
}

func (n *TestNetwork) FlushLog() {
	for _, l := range n.log {
		l.Flush()
	}
}
