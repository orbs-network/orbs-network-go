package commands

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/devtools/jsonapi"
	"io/ioutil"
	"os"
)

func ShowDeployUsage() {
	fmt.Println("Usage:  $ gamma-cli deploy MyContractName path/to/contract.go")
	os.Exit(2)
}

func HandleDeployCommand(args []string) {
	if len(args) < 2 {
		ShowDeployUsage()
	}

	contractName := args[0]
	pathToCodeFile := args[1]

	_, err := os.Stat(pathToCodeFile)

	if err != nil {
		fmt.Println("Could not find contract source code at the provided path")
		fmt.Println(pathToCodeFile)
		os.Exit(1)
	}

	codeBytes, err := ioutil.ReadFile(pathToCodeFile)
	if err != nil {
		fmt.Println("Could not load Go source code", err)
		os.Exit(1)
	}

	argName := jsonapi.MethodArgument{
		Name:  "serviceName",
		Type:  "string",
		Value: contractName,
	}

	argProcessor := jsonapi.MethodArgument{
		Name:  "processorType",
		Type:  "uint32",
		Value: uint32(1), // Native processor - for Go contracts
	}

	codeInHex := hex.EncodeToString(codeBytes)

	argCode := jsonapi.MethodArgument{
		Name:  "code",
		Type:  "bytes",
		Value: codeInHex,
	}

	deployTx := jsonapi.Transaction{
		ContractName: "_Deployments",
		MethodName:   "deployService",
		Arguments:    []jsonapi.MethodArgument{argName, argProcessor, argCode},
	}

	jsonBytes, err := json.Marshal(deployTx)

	err = ioutil.WriteFile("./.deploy.json", jsonBytes, 0644)
	if err != nil {
		fmt.Println("Could not write deployment action json", err)
		os.Exit(1)
	}

	runArgs := []string{"send", "./.deploy.json"}
	runArgs = append(runArgs, args[2:]...)

	HandleRunCommand(runArgs)
}
