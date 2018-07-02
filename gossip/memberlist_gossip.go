package main

import (
	"os"
	"strings"
	"fmt"
	"strconv"
	"time"
	"github.com/hashicorp/memberlist"
)

type MemberlistGossipConfig struct {
	Name string
	Port int
	Peers []string
}

func NewList(config MemberlistGossipConfig) *memberlist.Memberlist {
	fmt.Println("Creating memberlist with config", config)

	listConfig := memberlist.DefaultLocalConfig()
	listConfig.BindPort = config.Port
	listConfig.Name = config.Name

	list, err := memberlist.Create(listConfig)
	if err != nil {
		panic("Failed to create memberlist: " + err.Error())
	}
	
	// Join an existing cluster by specifying at least one known member.
	n, err := list.Join(config.Peers)

	if err != nil {
		fmt.Println("Failed to join cluster: " + err.Error())
	} else {
		fmt.Println("Connected to", n, "hosts")
	}

	return list
}

func PrintPeers(list *memberlist.Memberlist) {
	// Ask for members of the cluster
	for _, member := range list.Members() {
		fmt.Printf("Member: %s %s\n", member.Name, member.Addr)
	}
}

func main() {
	port, _ := strconv.ParseInt(os.Getenv("GOSSIP_PORT"), 10, 0)
	nodeName := os.Getenv("NODE_NAME")
	peers := strings.Split(os.Getenv("GOSSIP_PEERS"), ",")

	config := MemberlistGossipConfig{nodeName, int(port), peers}
	list := NewList(config)

	for {
		go PrintPeers(list)
		time.Sleep(3 * time.Second)
	}
}
