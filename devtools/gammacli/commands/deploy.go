package commands

import (
	"encoding/hex"
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/devtools/gammacli"
	"io/ioutil"
	"os"
)

func ShowDeployUsage() string {
	return "Usage:  $ gamma-cli deploy MyContractName path/to/contract.go"
}

func HandleDeployCommand(args []string) (string, error) {
	if len(args) < 2 {
		return ShowDeployUsage(), nil
	}

	contractName := args[0]
	pathToCodeFile := args[1]

	_, err := os.Stat(pathToCodeFile)

	if err != nil {
		returnString := `
Could not find contract source code at the provided path
pathToCodeFile
`
		return returnString, nil
	}

	codeBytes, err := ioutil.ReadFile(pathToCodeFile)
	if err != nil {
		return "", err
	}

	argName := gammacli.JSONMethodArgument{
		Name:  "serviceName",
		Type:  "string",
		Value: contractName,
	}

	argProcessor := gammacli.JSONMethodArgument{
		Name:  "processorType",
		Type:  "uint32",
		Value: uint32(1), // Native processor - for Go contracts
	}

	codeInHex := hex.EncodeToString(codeBytes)

	argCode := gammacli.JSONMethodArgument{
		Name:  "code",
		Type:  "bytes",
		Value: codeInHex,
	}

	deployTx := gammacli.JSONTransaction{
		ContractName: "_Deployments",
		MethodName:   "deployService",
		Arguments:    []gammacli.JSONMethodArgument{argName, argProcessor, argCode},
	}

	jsonBytes, err := json.Marshal(deployTx)

	err = ioutil.WriteFile("./.deploy.json", jsonBytes, 0644)
	if err != nil {
		return "", err
	}

	runArgs := []string{"send", "./.deploy.json"}
	runArgs = append(runArgs, args[2:]...)

	return HandleRunCommand(runArgs)
}
