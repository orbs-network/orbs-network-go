package harness

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/test"
	harnessInstrumentation "github.com/orbs-network/orbs-network-go/test/harness/instrumentation"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type AcceptanceTestNetwork interface {
	FlushLog()
	GossipTransport() gossipAdapter.TamperingTransport
	LoopControl(nodeIndex int) harnessInstrumentation.BrakingLoop
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	SendTransfer(nodeIndex int, amount uint64) chan *client.SendTransactionResponse
	CallGetBalance(nodeIndex int) chan uint64
}

type acceptanceTestNetwork struct {
	nodes           []networkNode
	gossipTransport gossipAdapter.TamperingTransport
}

type networkNode struct {
	index            int
	config           config.NodeConfig
	log              harnessInstrumentation.BufferedLog
	latch            harnessInstrumentation.Latch
	loopControl      harnessInstrumentation.BrakingLoop
	blockPersistence blockStorageAdapter.InMemoryBlockPersistence
	statePersistence stateStorageAdapter.InMemoryStatePersistence
	nodeLogic        bootstrap.NodeLogic
}

func NewTestNetwork(numNodes uint32) AcceptanceTestNetwork {
	sharedTamperingTransport := gossipAdapter.NewTamperingTransport()
	nodes := make([]networkNode, numNodes)
	for i, _ := range nodes {
		nodes[i].index = i
		nodePublicKey := []byte{byte(i + 1)} // TODO: improve this to real generation of public key
		nodeName := fmt.Sprintf("node-pkey-%x", nodePublicKey)
		isLeader := (i == 0) // TODO: remove the concept of leadership
		nodes[i].config = config.NewHardCodedConfig(numNodes, nodePublicKey)
		nodes[i].log = harnessInstrumentation.NewBufferedLog(nodeName)
		nodes[i].latch = harnessInstrumentation.NewLatch()
		nodes[i].loopControl = harnessInstrumentation.NewBrakingLoop(nodes[i].log)
		nodes[i].blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence(nodes[i].config)
		nodes[i].statePersistence = stateStorageAdapter.NewInMemoryStatePersistence(nodes[i].config)
		nodes[i].nodeLogic = bootstrap.NewNodeLogic(
			sharedTamperingTransport,
			nodes[i].blockPersistence,
			nodes[i].statePersistence,
			instrumentation.NewCompositeReporting([]instrumentation.Reporting{nodes[i].log, nodes[i].latch}),
			nodes[i].loopControl,
			nodes[i].config,
			isLeader,
		)
	}
	return &acceptanceTestNetwork{
		nodes:           nodes,
		gossipTransport: sharedTamperingTransport,
	}
}

func (n *acceptanceTestNetwork) FlushLog() {
	for i, _ := range n.nodes {
		n.nodes[i].log.Flush()
	}
}

func (n *acceptanceTestNetwork) LoopControl(nodeIndex int) harnessInstrumentation.BrakingLoop {
	return n.nodes[nodeIndex].loopControl
}

func (n *acceptanceTestNetwork) GossipTransport() gossipAdapter.TamperingTransport {
	return n.gossipTransport
}

func (n *acceptanceTestNetwork) BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.nodes[nodeIndex].blockPersistence
}

func (n *acceptanceTestNetwork) SendTransfer(nodeIndex int, amount uint64) chan *client.SendTransactionResponse {
	ch := make(chan *client.SendTransactionResponse)
	go func() {
		request := (&client.SendTransactionRequestBuilder{
			SignedTransaction: test.TransferTransaction().WithAmount(amount).Builder(),
		}).Build()
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: request,
		})
		if err != nil {
			// TODO: handle error
		}
		ch <- output.ClientResponse
	}()
	return ch
}

func (n *acceptanceTestNetwork) CallGetBalance(nodeIndex int) chan uint64 {
	ch := make(chan uint64)
	go func() {
		request := (&client.CallMethodRequestBuilder{
			Transaction: &protocol.TransactionBuilder{
				ContractName: "BenchmarkToken",
				MethodName:   "getBalance",
			},
		}).Build()
		publicApi := n.nodes[nodeIndex].nodeLogic.PublicApi()
		output, err := publicApi.CallMethod(&services.CallMethodInput{
			ClientRequest: request,
		})
		if err != nil {
			// TODO: handle error
		}
		ch <- output.ClientResponse.OutputArgumentsIterator().NextOutputArguments().Uint64Value()
	}()
	return ch
}
