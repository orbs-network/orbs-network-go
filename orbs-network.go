package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	. "github.com/orbs-network/orbs-network-go/gossip"
)

func simulateConversation() {

}

func main() {
	port, _ := strconv.ParseInt(os.Getenv("GOSSIP_PORT"), 10, 0)
	nodeName := os.Getenv("NODE_NAME")
	peers := strings.Split(os.Getenv("GOSSIP_PEERS"), ",")

	config := MemberlistGossipConfig{nodeName, int(port), peers}
	gossip := NewGossip(config)

	for {
		go gossip.Join()
		go gossip.PrintPeers()
		time.Sleep(3 * time.Second)
	}
}
