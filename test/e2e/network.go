package e2e

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"os"
	"time"
)

var OwnerOfAllSupply = keys.Ed25519KeyPairForTests(5) // needs to be a constant across all e2e tests since we deploy the contract only once

const LOCAL_NETWORK_SIZE = 3

type inProcessE2ENetwork struct {
	nodes []bootstrap.Node
}

func NewInProcessE2ENetwork() *inProcessE2ENetwork {
	cleanNativeProcessorCache()
	cleanBlockStorage()

	return &inProcessE2ENetwork{bootstrapNetwork()}
}

func (h *inProcessE2ENetwork) GracefulShutdown() {
	for _, node := range h.nodes {
		node.GracefulShutdown(0) // meaning don't have a deadline timeout so allowing enough time for shutdown to free port
	}

	cleanNativeProcessorCache()
	cleanBlockStorage()

}

func bootstrapNetwork() (nodes []bootstrap.Node) {
	firstRandomPort := test.RandomPort()

	federationNodes := make(map[string]config.FederationNode)
	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
		nodeAddress := keys.EcdsaSecp256K1KeyPairForTests(i).NodeAddress()
		federationNodes[nodeAddress.KeyForMap()] = config.NewHardCodedFederationNode(nodeAddress)
		gossipPeers[nodeAddress.KeyForMap()] = config.NewHardCodedGossipPeer(firstRandomPort+i, "127.0.0.1")
	}

	ethereumEndpoint := os.Getenv("ETHEREUM_ENDPOINT") //TODO v1 unite how this config is fetched

	_ = os.MkdirAll(config.GetProjectSourceRootPath()+"/_logs", 0755)
	console := log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter())

	logger := log.GetLogger().WithTags(
		log.String("_test", "e2e"),
		log.String("_branch", os.Getenv("GIT_BRANCH")),
		log.String("_commit", os.Getenv("GIT_COMMIT"))).
		WithFilters(log.IgnoreMessagesMatching("Metric recorded")).
		WithOutput(console)
	leaderKeyPair := keys.EcdsaSecp256K1KeyPairForTests(0)
	for i := 0; i < LOCAL_NETWORK_SIZE; i++ {
		nodeKeyPair := keys.EcdsaSecp256K1KeyPairForTests(i)

		logFile, err := os.OpenFile(
			fmt.Sprintf("%s/_logs/node%d-%v.log", config.GetProjectSourceRootPath(), i+1, time.Now().Format(time.RFC3339Nano)),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0644)
		if err != nil {
			panic(err)
		}

		nodeLogger := logger.WithOutput(console, log.NewFormattingOutput(logFile, log.NewJsonFormatter()))
		processorArtifactPath, _ := getProcessorArtifactPath()

		cfg := config.
			ForE2E(
				processorArtifactPath,
				federationNodes,
				gossipPeers,
				leaderKeyPair.NodeAddress(),
				consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
				ethereumEndpoint).
			OverrideNodeSpecificValues(
				firstRandomPort+i,
				nodeKeyPair.NodeAddress(),
				nodeKeyPair.PrivateKey(),
				blockStorageDataDirPrefix)

		deployBlockStorageFiles(cfg.BlockStorageDataDir(), logger)

		node := bootstrap.NewNode(cfg, nodeLogger, fmt.Sprintf(":%d", START_HTTP_PORT+i))

		nodes = append(nodes, node)
	}
	return nodes
}
