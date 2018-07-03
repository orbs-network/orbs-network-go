package testharness

/*
Objects here are only for testing purposes, not to be used in real code
 */

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/loopcontrol"
	"github.com/orbs-network/orbs-network-go/publicapi"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/testharness/gossip"
	"github.com/orbs-network/orbs-network-go/config"
)

type AcceptanceTestNetwork interface {
	FlushLog()
	LeaderLoopControl() loopcontrol.BrakingLoop
	Gossip() gossip.PausableTransport
	Leader() publicapi.PublicApi
	Validator() publicapi.PublicApi
	LeaderBp() blockstorage.InMemoryBlockPersistence
	ValidatorBp() blockstorage.InMemoryBlockPersistence

	SendTransaction(gatewayNode publicapi.PublicApi, transaction *types.Transaction) chan interface{}
	CallMethod(node publicapi.PublicApi) chan int
}

type acceptanceTestNetwork struct {
	leader            bootstrap.Node
	validator         bootstrap.Node
	leaderLatch       events.Latch
	leaderBp          blockstorage.InMemoryBlockPersistence
	validatorBp       blockstorage.InMemoryBlockPersistence
	gossip            gossip.PausableTransport
	leaderLoopControl loopcontrol.BrakingLoop

	log []events.BufferedLog
}

func CreateTestNetwork() AcceptanceTestNetwork {
	leaderLog := events.NewBufferedLog("leader")
	leaderLatch := events.NewLatch()
	validatorLog := events.NewBufferedLog("validator")

	leaderLoopControl := loopcontrol.NewBrakingLoop(leaderLog)

	inMemoryGossip := gossip.NewPausableTransport()
	leaderBp := blockstorage.NewInMemoryBlockPersistence("leaderBp")
	validatorBp := blockstorage.NewInMemoryBlockPersistence("validatorBp")
	nodeConfig := config.NewHardCodedConfig(2)

	leader := bootstrap.NewNode(inMemoryGossip, leaderBp, events.NewCompositeEvents([]events.Events{leaderLog, leaderLatch}), leaderLoopControl, nodeConfig,true)
	validator := bootstrap.NewNode(inMemoryGossip, validatorBp, validatorLog, loopcontrol.NewBrakingLoop(validatorLog), nodeConfig,false)

	return &acceptanceTestNetwork{
		leader:            leader,
		validator:         validator,
		leaderLatch:       leaderLatch,
		leaderBp:          leaderBp,
		validatorBp:       validatorBp,
		gossip:            inMemoryGossip,
		leaderLoopControl: leaderLoopControl,

		log: []events.BufferedLog{leaderLog, validatorLog},
	}
}

func (n *acceptanceTestNetwork) FlushLog() {
	for _, l := range n.log {
		l.Flush()
	}
}

func (n *acceptanceTestNetwork) LeaderLoopControl() loopcontrol.BrakingLoop {
	return n.leaderLoopControl
}

func (n *acceptanceTestNetwork) Gossip() gossip.PausableTransport {
	return n.gossip
}

func (n *acceptanceTestNetwork) Leader() publicapi.PublicApi {
	return n.leader.GetPublicApi()
}

func (n *acceptanceTestNetwork) Validator() publicapi.PublicApi {
	return n.validator.GetPublicApi()
}

func (n *acceptanceTestNetwork) LeaderBp() blockstorage.InMemoryBlockPersistence {
	return n.leaderBp
}

func (n *acceptanceTestNetwork) ValidatorBp() blockstorage.InMemoryBlockPersistence {
	return n.validatorBp
}

func (n *acceptanceTestNetwork) SendTransaction(gatewayNode publicapi.PublicApi, transaction *types.Transaction) chan interface{} {
	ch := make(chan interface{})
	go func() {
		gatewayNode.SendTransaction(transaction)
		ch <- nil
	}()
	return ch
}

func (n *acceptanceTestNetwork) CallMethod(node publicapi.PublicApi) chan int {
	ch := make(chan int)
	go func() {
		ch <- node.CallMethod()
	}()
	return ch
}



