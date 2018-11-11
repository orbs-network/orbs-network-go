package main

import (
	"flag"
	"github.com/orbs-network/orbs-network-go/devtools/gammaserver"
	"strconv"
)

func main() {
	port := flag.Int("port", 8080, "The port to bind the gamma server to")
	flag.Parse()

	var serverAddress = ":" + strconv.Itoa(*port)

	// TODO Remove the blocking boolean flag from here and make it an environment variable. (Shouldn't concern our users who will see how we bootstrap this)
	gammaserver.StartGammaServer(serverAddress, true)
}
