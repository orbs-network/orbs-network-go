package main

import (
	"flag"
	"github.com/orbs-network/orbs-network-go/bootstrap/gamma"
	"strconv"
)

func main() {
	port := flag.Int("port", 8080, "The port to bind the gamma server to")
	flag.Parse()

	var serverAddress = ":" + strconv.Itoa(*port)

	// TODO(v1) add WaitUntilShutdown so this behaves like the regular main (no blocking flag)
	gamma.StartGammaServer(serverAddress, true)
}
