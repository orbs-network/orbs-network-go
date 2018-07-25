package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/test/builders"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type AcceptanceTestNetwork interface {
	FlushLog()
	GossipTransport() gossipAdapter.TamperingTransport
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	SendTransfer(nodeIndex int, amount uint64) chan *client.SendTransactionResponse
	SendInvalidTransfer(nodeIndex int) chan *client.SendTransactionResponse
	CallGetBalance(nodeIndex int) chan uint64
}

type acceptanceTestNetwork struct {
	nodes           []networkNode
	gossipTransport gossipAdapter.TamperingTransport
}

type networkNode struct {
	index            int
	config           config.NodeConfig
	blockPersistence blockStorageAdapter.InMemoryBlockPersistence
	statePersistence stateStorageAdapter.InMemoryStatePersistence
	nodeLogic        bootstrap.NodeLogic
}

func NewTestNetwork(ctx context.Context, numNodes uint32) AcceptanceTestNetwork {
	sharedTamperingTransport := gossipAdapter.NewTamperingTransport()
	nodes := make([]networkNode, numNodes)
	for i, _ := range nodes {
		nodes[i].index = i
		nodePublicKey := []byte{byte(i + 1)} // TODO: improve this to real generation of public key
		constantConsensusLeaderPublicKey := []byte{byte(1)}
		nodeName := fmt.Sprintf("node-pkey-%x", nodePublicKey)

		nodes[i].config = config.NewHardCodedConfig(
			numNodes,
			nodePublicKey,
			constantConsensusLeaderPublicKey,
			consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX,
		)

		nodes[i].blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence(nodes[i].config)
		nodes[i].statePersistence = stateStorageAdapter.NewInMemoryStatePersistence(nodes[i].config)

		nodes[i].nodeLogic = bootstrap.NewNodeLogic(
			ctx,
			sharedTamperingTransport,
			nodes[i].blockPersistence,
			nodes[i].statePersistence,
			instrumentation.GetLogger().For(instrumentation.Node(nodeName)),
			nodes[i].config,
		)

	}
	return &acceptanceTestNetwork{
		nodes:           nodes,
		gossipTransport: sharedTamperingTransport,
	}
}

func (n *acceptanceTestNetwork) FlushLog() {

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
			SignedTransaction: builders.TransferTransaction().WithAmount(amount).Builder(),
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

func (n *acceptanceTestNetwork) SendInvalidTransfer(nodeIndex int) chan *client.SendTransactionResponse {
	ch := make(chan *client.SendTransactionResponse)
	go func() {
		request := (&client.SendTransactionRequestBuilder{
			SignedTransaction: builders.TransferTransaction().WithInvalidContent().Builder(),
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
