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
	Leader       bootstrap.Node
	Validator    bootstrap.Node
	LeaderEvents events.Events
	LeaderBp     blockstorage.InMemoryBlockPersistence
	ValidatorBp  blockstorage.InMemoryBlockPersistence
	Gossip       gossip.PausableGossip
}

func CreateTestNetwork() TestNetwork {
	leaderEvents := events.NewEvents()
	inMemoryGossip := gossip.NewPausableGossip()
	leaderBp := blockstorage.NewInMemoryBlockPersistence("leaderBp")
	validatorBp := blockstorage.NewInMemoryBlockPersistence("validatorBp")

	leader := bootstrap.NewNode(inMemoryGossip, leaderBp, leaderEvents, true)
	validator := bootstrap.NewNode(inMemoryGossip, validatorBp, events.NewEvents(), false)
	inMemoryGossip.RegisterAll([]gossip.Listener{leader, validator})

	return TestNetwork{
		Leader:       leader,
		Validator:    validator,
		LeaderEvents: leaderEvents,
		LeaderBp:     leaderBp,
		ValidatorBp:  validatorBp,
		Gossip:       inMemoryGossip,
	}
}

