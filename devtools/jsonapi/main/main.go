package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi/main/commands"
	"os"
)

// gamma-cli start [-port=8080]
// gamma-cli run call|send [-public-key=<pubkey>] [-private-key=<privkey>] [-host=<http://....>]

func main() {
	if len(os.Args) < 2 {
		// TODO implement a welcome message here
		fmt.Println("must specify which command to run")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		commands.HandleRunCommand(os.Args[2:])
	case "start":
		commands.HandleStartCommand(os.Args[2:])
	case "deploy":
		commands.HandleDeployCommand(os.Args[2:])
	case "genKeys":
		commands.HandleGenKeysCommand()
	default:
		flag.PrintDefaults()
		os.Exit(1)
	}
}
