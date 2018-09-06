package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"io"
	"io/ioutil"
	"os"
	"strconv"
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

type peer struct {
	Key  string
	IP   string
	Port uint16
}

func getFederationNodes(logger log.BasicLogger, input string) map[string]config.FederationNode {
	federationNodes := make(map[string]config.FederationNode)

	if input == "" {
		return federationNodes
	}

	var peers []peer

	err := json.Unmarshal([]byte(input), &peers)
	if err != nil {
		logger.Error("Failed to parse peers configuration", log.Error(err))
		return federationNodes
	}

	for _, peer := range peers {
		publicKey, _ := hex.DecodeString(peer.Key)
		federationNodes[string(publicKey)] = config.NewHardCodedFederationNode(publicKey, peer.Port, peer.IP)
	}

	return federationNodes
}

func main() {
	// TODO: change this to a config like HardCodedConfig that takes config from env or json
	port, _ := strconv.ParseInt(os.Getenv("PORT"), 10, 0)
	nodePublicKey, _ := hex.DecodeString(os.Getenv("NODE_PUBLIC_KEY"))
	nodePrivateKey, _ := hex.DecodeString(os.Getenv("NODE_PRIVATE_KEY"))
	federationNodes := os.Getenv("FEDERATION_NODES")
	consensusLeader, _ := hex.DecodeString(os.Getenv("CONSENSUS_LEADER"))
	httpAddress := ":" + strconv.FormatInt(port, 10)
	logPath := os.Getenv("LOG_PATH")
	silentLog := os.Getenv("SILENT") == "true"

	logger := getLogger(logPath, silentLog)

	// TODO: move this code to the config we decided to add, the HardCodedConfig stuff is just placeholder

	peers := getFederationNodes(logger, federationNodes)

	bootstrap.NewNode(
		httpAddress,
		nodePublicKey,
		nodePrivateKey,
		peers,
		consensusLeader,
		consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS,
		logger,
	).WaitUntilShutdown()
}
