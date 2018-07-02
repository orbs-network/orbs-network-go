package gossip

import (
	"fmt"
	"github.com/hashicorp/memberlist"
)

type MemberlistGossipConfig struct {
	Name string
	Port int
	Peers []string
}

func NewGossip(config MemberlistGossipConfig) *memberlist.Memberlist {
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
