package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/bootstrap/gamma"
	"github.com/orbs-network/orbs-network-go/config"
	"strconv"
)

func main() {
	port := flag.Int("port", 8080, "The port to bind the gamma server to")
	profiling := flag.Bool("profiling", false, "enable profiling")
	version := flag.Bool("version", false, "returns information about version")

	flag.Parse()

	if *version {
		fmt.Println(config.GetVersion())
		return
	}

	var serverAddress = ":" + strconv.Itoa(*port)

	// TODO(v1) add WaitUntilShutdown so this behaves like the regular main (no blocking flag)
	gamma.StartGammaServer(serverAddress, *profiling, true)
}
