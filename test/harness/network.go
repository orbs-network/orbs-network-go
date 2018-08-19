package harness

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	blockStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/blockstorage/adapter"
	gossipAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	stateStorageAdapter "github.com/orbs-network/orbs-network-go/test/harness/services/statestorage/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"os"
	"testing"
)

func WithNetwork(t *testing.T, numNodes uint32, consensusAlgos []consensus.ConsensusAlgoType, f func(network AcceptanceTestNetwork)) {
	for _, consensusAlgo := range consensusAlgos {
		test.WithContext(func(ctx context.Context) {
			network := NewTestNetwork(ctx, numNodes, consensusAlgo)
			f(network)
			if t.Failed() { // avoid serializing state if test succeeded
				network.DumpState()
			}
		})
	}
}

func WithAlgos(algos ...consensus.ConsensusAlgoType) []consensus.ConsensusAlgoType {
	return algos
}

type AcceptanceTestNetwork interface {
	Description() string
	DeployBenchmarkToken()
	GossipTransport() gossipAdapter.TamperingTransport
	BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence
	SendTransfer(nodeIndex int, amount uint64) chan *client.SendTransactionResponse
	SendInvalidTransfer(nodeIndex int) chan *client.SendTransactionResponse
	CallGetBalance(nodeIndex int) chan uint64
	DumpState()
	WaitForTransactionInState(nodeIndex int, txhash primitives.Sha256)
}

type acceptanceTestNetwork struct {
	nodes           []networkNode
	gossipTransport gossipAdapter.TamperingTransport
	description     string
}

type networkNode struct {
	index            int
	config           config.NodeConfig
	blockPersistence blockStorageAdapter.InMemoryBlockPersistence
	statePersistence stateStorageAdapter.InMemoryStatePersistence
	nodeLogic        bootstrap.NodeLogic
}

func NewTestNetwork(ctx context.Context, numNodes uint32, consensusAlgo consensus.ConsensusAlgoType) AcceptanceTestNetwork {

	testLogger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	testLogger.Info("===========================================================================")
	testLogger.Info("creating acceptance test network", log.String("consensus", consensusAlgo.String()), log.Uint32("num-nodes", numNodes))
	description := fmt.Sprintf("network with %d nodes running %s", numNodes, consensusAlgo)

	sharedTamperingTransport := gossipAdapter.NewTamperingTransport()
	leaderKeyPair := keys.Ed25519KeyPairForTests(0)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < int(numNodes); i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey)
	}

	nodes := make([]networkNode, numNodes)
	for i := range nodes {
		nodes[i].index = i
		nodeKeyPair := keys.Ed25519KeyPairForTests(i)
		nodeName := fmt.Sprintf("%s", nodeKeyPair.PublicKey()[:3])

		nodes[i].config = config.ForAcceptanceTests(
			federationNodes,
			nodeKeyPair.PublicKey(),
			nodeKeyPair.PrivateKey(),
			leaderKeyPair.PublicKey(),
			consensusAlgo,
		)

		nodes[i].statePersistence = stateStorageAdapter.NewInMemoryStatePersistence()
		nodes[i].blockPersistence = blockStorageAdapter.NewInMemoryBlockPersistence()

		nodes[i].nodeLogic = bootstrap.NewNodeLogic(
			ctx,
			sharedTamperingTransport,
			nodes[i].blockPersistence,
			nodes[i].statePersistence,
			testLogger.For(log.Node(nodeName)),
			nodes[i].config,
		)
	}

	return &acceptanceTestNetwork{
		nodes:           nodes,
		gossipTransport: sharedTamperingTransport,
		description:     description,
	}
}

func (n *acceptanceTestNetwork) WaitForTransactionInState(nodeIndex int, txhash primitives.Sha256) {
	blockHeight := n.BlockPersistence(nodeIndex).WaitForTransaction(txhash)
	err := n.nodes[nodeIndex].statePersistence.WaitUntilCommittedBlockOfHeight(blockHeight)
	if err != nil {
		panic(fmt.Sprintf("statePersistence.WaitUntilCommittedBlockOfHeight failed: %s", err.Error()))
	}
}

func (n *acceptanceTestNetwork) Description() string {
	return n.description
}

func (n *acceptanceTestNetwork) GossipTransport() gossipAdapter.TamperingTransport {
	return n.gossipTransport
}

func (n *acceptanceTestNetwork) BlockPersistence(nodeIndex int) blockStorageAdapter.InMemoryBlockPersistence {
	return n.nodes[nodeIndex].blockPersistence
}

func (n *acceptanceTestNetwork) DeployBenchmarkToken() {
	tx := <-n.SendTransfer(0, 0) // deploy BenchmarkToken by running an empty transaction
	for i := range n.nodes {
		n.WaitForTransactionInState(i, tx.TransactionReceipt().Txhash())
	}
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
			panic(fmt.Sprintf("error in transfer: %v", err)) // TODO: improve
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
			panic(fmt.Sprintf("error in invalid transfer: %v", err)) // TODO: improve
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
			panic(fmt.Sprintf("error in get balance: %v", err)) // TODO: improve
		}
		ch <- output.ClientResponse.OutputArgumentsIterator().NextOutputArguments().Uint64Value()
	}()
	return ch
}

func (n *acceptanceTestNetwork) DumpState() {
	testLogger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	for i := range n.nodes {
		testLogger.Info("state dump", log.Int("node", i), log.String("data", n.nodes[i].statePersistence.Dump()))
	}
}
