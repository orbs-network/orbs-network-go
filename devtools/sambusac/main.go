package main

import (
	"flag"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi"
	"strconv"
)

func main() {
	port := flag.Int("port", 8080, "The port to bind the Sambusac server to")
	flag.Parse()

	var serverAddress = ":" + strconv.Itoa(*port)
	var pathToContracts = "."

	// TODO Remove the blocking boolean flag from here and make it an environment variable. (Shouldn't concern our users who will see how we bootstrap this)
	jsonapi.StartSambusac(serverAddress, pathToContracts, true)
}
