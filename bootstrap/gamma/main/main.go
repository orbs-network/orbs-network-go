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

	// TODO Remove the blocking boolean flag from here and make it an environment variable. (Shouldn't concern our users who will see how we bootstrap this)
	gamma.StartGammaServer(serverAddress, true)
}
