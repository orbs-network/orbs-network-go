package main

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func getLogger(path string, silent bool) log.BasicLogger {
	if path == "" {
		path = "./orbs-network.log"
	}

	logFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	var stdout io.Writer
	stdout = os.Stdout

	if silent {
		stdout = ioutil.Discard
	}

	stdoutOutput := log.NewOutput(stdout).WithFormatter(log.NewHumanReadableFormatter())
	fileOutput := log.NewOutput(logFile)

	return log.GetLogger().WithOutput(stdoutOutput, fileOutput)
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
	silentLog := os.Getenv("SILENT") == "true"

	logger := getLogger(logPath, silentLog)

	// TODO: move this code to the config we decided to add, the HardCodedConfig stuff is just placeholder
	federationNodes := make(map[string]config.FederationNode)
	for _, federationNodePublicKey := range federationNodePublicKeys {
		publicKey, _ := hex.DecodeString(federationNodePublicKey)
		federationNodes[string(publicKey)] = config.NewHardCodedFederationNode(publicKey)
	}

	// TODO: change MemberlistGossipConfig to the standard config mechanism
	config := gossipAdapter.MemberlistGossipConfig{nodePublicKey, int(gossipPort), peers}
	gossipTransport := gossipAdapter.NewMemberlistTransport(config)

	bootstrap.NewNode(
		httpAddress,
		nodePublicKey,
		nodePrivateKey,
		federationNodes,
		consensusLeader,
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
		logger,
		gossipTransport,
	).WaitUntilShutdown()
}
