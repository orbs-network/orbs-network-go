package main

import (
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"os"
	"strconv"
	"strings"
)

func main() {
	port, _ := strconv.ParseInt(os.Getenv("PORT"), 10, 0)
	gossipPort, _ := strconv.ParseInt(os.Getenv("GOSSIP_PORT"), 10, 0)
	nodePublicKey, _ := hex.DecodeString(os.Getenv("NODE_PUBLIC_KEY"))
	peers := strings.Split(os.Getenv("GOSSIP_PEERS"), ",")
	consensusLeader, _ := hex.DecodeString(os.Getenv("CONSENSUS_LEADER"))
	httpAddress := ":" + strconv.FormatInt(port, 10)

	// TODO: change this to new config mechanism
	config := gossipAdapter.MemberlistGossipConfig{nodePublicKey, int(gossipPort), peers}
	gossipTransport := gossipAdapter.NewMemberlistTransport(config)

	bootstrap.NewNode(
		httpAddress,
		nodePublicKey,
		3,
		consensusLeader,
		consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX,
		gossipTransport,
	).WaitUntilShutdown()
}
