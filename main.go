package main

import "github.com/orbs-network/orbs-network-go/bootstrap"

func main() {
	bootstrap.NewHttpServer(":8080", "node1", true)

}
