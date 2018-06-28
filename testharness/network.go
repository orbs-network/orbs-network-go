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

	log []events.BufferingEvents
}

func CreateTestNetwork() TestNetwork {
	leaderLog := events.NewBufferingEvents("leader")
	leaderLatch := events.NewLatch()
	validatorLog := events.NewBufferingEvents("validator")

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

		log: []events.BufferingEvents{leaderLog, validatorLog},
	}
}

func (n *TestNetwork) FlushLog() {
	println("Flushing all BufferedEvents log entries")
	for _, l := range n.log {
		l.Flush()
	}
}
