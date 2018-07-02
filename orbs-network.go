package main

import (
  "os"
  "strings"
  "strconv"
  "time"
  . "github.com/orbs-network/orbs-network-go/gossip"
)

func main() {
	port, _ := strconv.ParseInt(os.Getenv("GOSSIP_PORT"), 10, 0)
	nodeName := os.Getenv("NODE_NAME")
	peers := strings.Split(os.Getenv("GOSSIP_PEERS"), ",")

	config := MemberlistGossipConfig{nodeName, int(port), peers}
	list := NewGossip(config)

	for {
		go PrintPeers(list)
		time.Sleep(3 * time.Second)
	}
}

