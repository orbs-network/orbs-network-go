package harness

/*
Objects here are only for testing purposes, not to be used in real code
 */

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/test/harness/gossip"
	"github.com/orbs-network/orbs-network-go/config"
	testinstrumentation "github.com/orbs-network/orbs-network-go/test/harness/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
)

type AcceptanceTestNetwork interface {
	FlushLog()
	LeaderLoopControl() testinstrumentation.BrakingLoop
	Gossip() gossip.TemperingTransport
	Leader() services.PublicApi
	Validator() services.PublicApi
	LeaderBp() blockstorage.InMemoryBlockPersistence
	ValidatorBp() blockstorage.InMemoryBlockPersistence

	Transfer(gatewayNode services.PublicApi, amount uint64) chan interface{}
	GetBalance(node services.PublicApi) chan uint64
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

func (n *acceptanceTestNetwork) Leader() services.PublicApi {
	return n.leader.GetPublicApi()
}

func (n *acceptanceTestNetwork) Validator() services.PublicApi {
	return n.validator.GetPublicApi()
}

func (n *acceptanceTestNetwork) LeaderBp() blockstorage.InMemoryBlockPersistence {
	return n.leaderBp
}

func (n *acceptanceTestNetwork) ValidatorBp() blockstorage.InMemoryBlockPersistence {
	return n.validatorBp
}

func (n *acceptanceTestNetwork) Transfer(gatewayNode services.PublicApi, amount uint64) chan interface{} {
	ch := make(chan interface{})
	go func() {

		tx := &protocol.SignedTransactionBuilder{TransactionContent: &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "transfer",
			InputArgument: []*protocol.MethodArgumentBuilder{
				{Name: "amount", Type: protocol.MethodArgumentTypeUint64, Uint64: amount},
			},
		}}
		input := &services.SendTransactionInput{ClientRequest: (&client.SendTransactionRequestBuilder{SignedTransaction: tx}).Build()}
		gatewayNode.SendTransaction(input)
		ch <- nil
	}()
	return ch
}

func (n *acceptanceTestNetwork) GetBalance(node services.PublicApi) chan uint64 {
	ch := make(chan uint64)
	go func() {
		cm := &protocol.TransactionBuilder{
			ContractName: "MelangeToken",
			MethodName:   "getBalance",
		}
		input := &services.CallMethodInput{ClientRequest: (&client.CallMethodRequestBuilder{Transaction:cm}).Build()}
		output, _ := node.CallMethod(input)
		ch <- output.ClientResponse.OutputArgumentIterator().NextOutputArgument().TypeUint64()
	}()
	return ch
}



