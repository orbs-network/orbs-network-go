package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/services/statestorage/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

func WithNetwork(numNodes uint32, consensusAlgos []consensus.ConsensusAlgoType, f func(network AcceptanceTestNetwork)) {
	for _, consensusAlgo := range consensusAlgos {
		test.WithContext(func(ctx context.Context) {
			network := NewTestNetwork(ctx, numNodes, consensusAlgo)
			f(network)
		})
	}
}

func WithAlgos(algos ...consensus.ConsensusAlgoType) []consensus.ConsensusAlgoType {
	return algos
}

type AcceptanceTestNetwork interface {
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
	statePersistence stateStorageAdapter.StatePersistence
	nodeLogic        bootstrap.NodeLogic
}

func NewTestNetwork(ctx context.Context, numNodes uint32, consensusAlgo consensus.ConsensusAlgoType) AcceptanceTestNetwork {

	testLogger := instrumentation.GetLogger().WithFormatter(instrumentation.NewHumanReadableFormatter())
	testLogger.Info("creating acceptance test network", instrumentation.String("consensus", consensusAlgo.String()), instrumentation.Uint32("num-nodes", numNodes))

	sharedTamperingTransport := gossipAdapter.NewTamperingTransport()
	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	}

	nodes := make([]networkNode, numNodes)
	for i, _ := range nodes {
		nodes[i].index = i
		nodeKeyPair := keys.Ed25519KeyPairForTests(i)
		nodeName := fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])

		nodes[i].config = config.NewHardCodedConfig(
			federationNodes,
			nodeKeyPair.PublicKey(),
			nodeKeyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
			1,
		)

		nodes[i].blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence(nodes[i].config)
		nodes[i].statePersistence = stateStorageAdapter.NewInMemoryStatePersistence(nodes[i].config)

		nodes[i].nodeLogic = bootstrap.NewNodeLogic(
			ctx,
			sharedTamperingTransport,
			nodes[i].blockPersistence,
			nodes[i].statePersistence,
			instrumentation.GetLogger().For(instrumentation.Node(nodeName)).WithFormatter(instrumentation.NewHumanReadableFormatter()),
			nodes[i].config,
		)
	}

	return &acceptanceTestNetwork{
		nodes:           nodes,
		gossipTransport: sharedTamperingTransport,
	}
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
