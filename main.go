package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	gossipAdapter "github.com/orbs-network/orbs-network-go/services/gossip/adapter"
)

func main() {
	port, _ := strconv.ParseInt(os.Getenv("PORT"), 10, 0)
	gossipPort, _ := strconv.ParseInt(os.Getenv("GOSSIP_PORT"), 10, 0)
	nodeName := os.Getenv("NODE_NAME")
	peers := strings.Split(os.Getenv("GOSSIP_PEERS"), ",")
	isLeader := os.Getenv("LEADER") == "true"

	config := gossipAdapter.memberlistGossipConfig{nodeName, int(gossipPort), peers}
	gossipTransport := gossipAdapter.NewMemberlistTransport(config)

	fmt.Println("PORT", port)

	bootstrap.NewNode(":"+strconv.FormatInt(port, 10), nodeName, gossipTransport, isLeader, 3)

	for {
		go gossipTransport.Join()
		// go gossip.PrintPeers()
		// go gossip.SendMessage("hello from " + nodeName + " " + time.Now().Format(time.RFC3339))
		time.Sleep(3 * time.Second)
	}
}
