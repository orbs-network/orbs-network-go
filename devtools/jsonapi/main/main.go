package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi/commands"
	"os"
)

// gamma-cli start [-port=8080]
// gamma-cli run call|send [-public-key=<pubkey>] [-private-key=<privkey>] [-host=<http://....>]

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Welcome to gamma-cli")
		fmt.Println("Example usage:")
		fmt.Println("")
		fmt.Println("$ gamma-cli start")
		fmt.Println("  Start gamma-server with 3 Orbs virtual nodes")
		fmt.Println("")
		fmt.Println("$ gamma-cli deploy MyContractName path/to/some/contract.go")
		fmt.Println("  Deploy your contract code onto the running blockchain on your local machine")
		fmt.Println("")
		fmt.Println("$ gamma-cli run send path/to/operation.json")
		fmt.Println("  Perform a contract method which mutates state")
		fmt.Println("")
		fmt.Println("$ gamma-cli run call path/to/operation.json")
		fmt.Println("  Perform a contract method which reads from state")
		fmt.Println("")
		os.Exit(0)
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
