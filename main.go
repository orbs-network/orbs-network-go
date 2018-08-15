package main

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"os"
	"strconv"
	"strings"
)

func getLogger(path string) instrumentation.BasicLogger {
	if path == "" {
		path = "./orbs-network.log"
	}

	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	stdoutOutput := instrumentation.NewOutput(os.Stdout).WithFormatter(instrumentation.NewHumanReadableFormatter())
	fileOutput := instrumentation.NewOutput(logFile)

	return instrumentation.GetLogger().WithOutput(stdoutOutput, fileOutput)
}

func main() {
	// TODO: change this to a config like HardCodedConfig that takes config from env or json
	port, _ := strconv.ParseInt(os.Getenv("PORT"), 10, 0)
	gossipPort, _ := strconv.ParseInt(os.Getenv("GOSSIP_PORT"), 10, 0)
	nodePublicKey, _ := hex.DecodeString(os.Getenv("NODE_PUBLIC_KEY"))
	nodePrivateKey, _ := hex.DecodeString(os.Getenv("NODE_PRIVATE_KEY"))
	peers := strings.Split(os.Getenv("GOSSIP_PEERS"), ",")
	federationNodePublicKeys := strings.Split(os.Getenv("FEDERATION_NODES"), ",")
	consensusLeader, _ := hex.DecodeString(os.Getenv("CONSENSUS_LEADER"))
	httpAddress := ":" + strconv.FormatInt(port, 10)
	logPath := os.Getenv("LOG_PATH")

	logger := getLogger(logPath)

	// TODO: move this code to the config we decided to add, the HardCodedConfig stuff is just placeholder
	federationNodes := make(map[string]config.FederationNode)
	for _, federationNodePublicKey := range federationNodePublicKeys {
		publicKey, _ := hex.DecodeString(federationNodePublicKey)
		federationNodes[string(nodePublicKey)] = config.NewHardCodedFederationNode(publicKey)
	}

	// TODO: change MemberlistGossipConfig to the standard config mechanism
	config := gossipAdapter.MemberlistGossipConfig{nodePublicKey, int(gossipPort), peers}
	gossipTransport := gossipAdapter.NewMemberlistTransport(config)

	bootstrap.NewNode(
		httpAddress,
		nodePublicKey,
		nodePrivateKey,
		federationNodes,
		70,
		5,
		5,
		30*60,
		consensusLeader,
		consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX,
		logger,
		2*1000,
		gossipTransport,
		5,
		3,
		300,
		300,
		2,
	).WaitUntilShutdown()
}
