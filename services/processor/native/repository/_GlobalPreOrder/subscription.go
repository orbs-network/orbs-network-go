package globalpreorder_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
)

func approve() {
	problem := _readSubscriptionProblem()
	if len(problem) != 0 {
		panic(problem)
	}
}

func refreshSubscription(ethContractAddress string) {
	currentContract := _readSubscriptionContract()

	if len(currentContract) != 0 && currentContract != ethContractAddress {
		panic(fmt.Sprintf("can only refresh current contract %s", currentContract))
	}

	if len(currentContract) == 0 {
		_writeSubscriptionContract(ethContractAddress)
	}

	jsonAbi := `
	[
    {
      "constant": true,
      "inputs": [],
      "name": "checkSubscription",
      "outputs": [
        {
          "name": "",
          "type": "string"
        }
      ],
      "payable": false,
      "stateMutability": "pure",
      "type": "function"
    }
  ]
	`
	var output string
	ethereum.CallMethod(ethContractAddress, jsonAbi, "checkSubscription", &output)
	if len(output) == 0 {
		_clearSubscriptionProblem()
	} else {
		_writeSubscriptionProblem(output)
	}
}

func _readSubscriptionProblem() string {
	return state.ReadString([]byte("SubscriptionProblem"))
}

func _writeSubscriptionProblem(problemStatus string) {
	state.WriteString([]byte("SubscriptionProblem"), problemStatus)
}

func _clearSubscriptionProblem() {
	state.Clear([]byte("SubscriptionProblem"))
}

func _readSubscriptionContract() string {
	return state.ReadString([]byte("SubscriptionContract"))
}

func _writeSubscriptionContract(ethContractAddress string) {
	state.WriteString([]byte("SubscriptionContract"), ethContractAddress)
}
