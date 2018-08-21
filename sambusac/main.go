package main

import (
	"github.com/orbs-network/orbs-network-go/jsonapi"
)

func main() {
	var serverAddress = ":8675"
	var pathToContracts = "."

	// TODO Remove the blocking boolean flag from here and make it an environment variable. (Shouldn't concern our users who will see how we bootstrap this)
	jsonapi.StartSambusac(serverAddress, pathToContracts, true)
}
