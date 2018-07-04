package main

import (
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	nodeId := os.Getenv("NODE_ID")

	//TODO system doesn't work because it doesn't block until shut down
	bootstrap.NewNode(":" +port, nodeId, true, 1)

	//TODO sigterm should call graceful shutdown
}
