package harness

/*
Objects here are only for testing purposes, not to be used in real code
 */

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/publicapi"
	"github.com/orbs-network/orbs-network-go/types"
	"github.com/orbs-network/orbs-network-go/test/harness/gossip"
	"github.com/orbs-network/orbs-network-go/config"
	testinstrumentation "github.com/orbs-network/orbs-network-go/test/harness/instrumentation"
)

type AcceptanceTestNetwork interface {
	FlushLog()
	LeaderLoopControl() testinstrumentation.BrakingLoop
	Gossip() gossip.TemperingTransport
	Leader() publicapi.PublicApi
	Validator() publicapi.PublicApi
	LeaderBp() blockstorage.InMemoryBlockPersistence
	ValidatorBp() blockstorage.InMemoryBlockPersistence

	SendTransaction(gatewayNode publicapi.PublicApi, transaction *types.Transaction) chan interface{}
	CallMethod(node publicapi.PublicApi) chan int
}

type acceptanceTestNetwork struct {
	leader            bootstrap.NodeLogic
	validator         bootstrap.NodeLogic
	leaderLatch       testinstrumentation.Latch
	leaderBp          blockstorage.InMemoryBlockPersistence
	validatorBp       blockstorage.InMemoryBlockPersistence
	gossip            gossip.TemperingTransport
	leaderLoopControl testinstrumentation.BrakingLoop

	log []testinstrumentation.BufferedLog
}

func CreateTestNetwork() AcceptanceTestNetwork {
	leaderConfig := config.NewHardCodedConfig(2, "leader")
	validatorConfig := config.NewHardCodedConfig(2, "validator")

	leaderLog := testinstrumentation.NewBufferedLog("leader")
	leaderLatch := testinstrumentation.NewLatch()
	validatorLog := testinstrumentation.NewBufferedLog("validator")

	leaderLoopControl := testinstrumentation.NewBrakingLoop(leaderLog)

	inMemoryGossip := gossip.NewTemperingTransport()
	leaderBp := blockstorage.NewInMemoryBlockPersistence(leaderConfig)
	validatorBp := blockstorage.NewInMemoryBlockPersistence(validatorConfig)

	leader := bootstrap.NewNodeLogic(inMemoryGossip, leaderBp, instrumentation.NewCompositeReporting([]instrumentation.Reporting{leaderLog, leaderLatch}), leaderLoopControl, leaderConfig, true)
	validator := bootstrap.NewNodeLogic(inMemoryGossip, validatorBp, validatorLog, testinstrumentation.NewBrakingLoop(validatorLog), validatorConfig, false)

	return &acceptanceTestNetwork{
		leader:            leader,
		validator:         validator,
		leaderLatch:       leaderLatch,
		leaderBp:          leaderBp,
		validatorBp:       validatorBp,
		gossip:            inMemoryGossip,
		leaderLoopControl: leaderLoopControl,

		log: []testinstrumentation.BufferedLog{leaderLog, validatorLog},
	}
}

func (n *acceptanceTestNetwork) FlushLog() {
	for _, l := range n.log {
		l.Flush()
	}
}

func (n *acceptanceTestNetwork) LeaderLoopControl() testinstrumentation.BrakingLoop {
	return n.leaderLoopControl
}

func (n *acceptanceTestNetwork) Gossip() gossip.TemperingTransport {
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



