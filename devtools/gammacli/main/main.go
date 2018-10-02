package main

import (
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/gammacli/commands"
	"os"
)

// gamma-cli start [-port=8080]
// gamma-cli deploy Counter path/to/your/code.go
// gamma-cli run call|send [-public-key=<pubkey>] [-private-key=<privkey>] [-host=<http://....>]

func main() {
	if len(os.Args) < 2 {
		fmt.Println(
			`
Gamma by ORBS v0.5
Gamma is a local ORBS blockchain to empower developers to easily and efficiently deploy, run & test smart contracts. 
Gamma runs an in-memory virtual chain on top of an ORBS blockchain with N nodes on your local machine.
Gamma-cli - the command line interface is deigned to help you to interact with the virtual chain.

Commands supported:

$ gamma-cli start 
  starts a local virtual chain over ORBS blockchain network, running on 3 nodes. 

$ gamma-cli deploy [contract name] [contract file] 
   Compile the smart contract with go v10.0 and deploy it on the personal orbs blockchain on your machine. 
   Example:  
   cd "$GOPATH/src/github.com/orbs-network/orbs-contract-sdk/" gamma-cli deploy Counter ./go/examples/counter/counter.go 

$ gamma-cli run  send [json file] 
  Use send when you want to send a transaction to a smart contract method that may
  change the the contract state. The transaction will be added to the blockchain under consensus.
   Example: gamma-cli run send ./go/examples/counter/jsons/add.json  

$ gamma-cli run call [json file] Use  when you want to access a smart contract method that reads from your state
  variables. In this case, the read is done on a local node, without undergoing consensus. 
  Example: gamma-cli run call ./go/examples/counter/jsons/get.json 

$ gamma-cli genKeys generates a new pair public and private key to sign on the transactions you send or your
  contract sends. The keys are stored on your computer on a file named .orbsKeys

For more information : https://github.com/orbs-network/orbs-contract-sdk.
Enjoy!
	`)
		os.Exit(0)
	}

	var exitCode int

	switch os.Args[1] {
	case "run":
		exitCode = commands.HandleRunCommand(os.Args[2:])
	case "start":
		exitCode = commands.HandleStartCommand(os.Args[2:])
	case "deploy":
		exitCode = commands.HandleDeployCommand(os.Args[2:])
	case "genKeys":
		exitCode = commands.HandleGenKeysCommand()
	default:
		flag.PrintDefaults()
	}

	os.Exit(exitCode)
}
