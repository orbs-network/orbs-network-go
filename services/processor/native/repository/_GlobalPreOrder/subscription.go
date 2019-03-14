package globalpreorder_systemcontract

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/env"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

func approve() {
	problem := _readSubscriptionProblem()
	if len(problem) != 0 {
		panic(problem)
	}
}

var plans = map[string]int{
	"B0": 450,
	"B1": 850,
	"B2": 1650,
	"B3": 3300,
	"B4": 6600,
	"B5": 13200,
}

func refreshSubscription(ethContractAddress string) {
	currentContract := _readSubscriptionContract()

	if len(currentContract) != 0 && currentContract != ethContractAddress {
		panic(fmt.Sprintf("can only refresh current contract %s", currentContract))
	}

	if len(currentContract) == 0 {
		_writeSubscriptionContract(ethContractAddress)
	}

	jsonAbi := `[{"constant":true,"inputs":[{"name":"_id","type":"bytes32"}],"name":"getSubscriptionData","outputs":[{"name":"id","type":"bytes32"},{"name":"profile","type":"string"},{"name":"startTime","type":"uint256"},{"name":"tokens","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`

	plan := ""
	id := [32]byte{}
	startTime := big.NewInt(0)
	tokens := big.NewInt(0)

	var output [4]interface{}
	output[0] = &id // id
	output[1] = &plan
	output[2] = &startTime
	output[3] = &tokens

	var vcid [32]byte
	binary.BigEndian.PutUint32(vcid[28:], 42) //TODO get actual virtual chain id
	ethereum.CallMethod(ethContractAddress, jsonAbi, "getSubscriptionData", &output, vcid)

	if err := _validateSubscription(vcid, id, plan, startTime, tokens); err == nil {
		_clearSubscriptionProblem()
	} else {
		_writeSubscriptionProblem(err.Error())
	}
}

func _validateSubscription(vcid [32]byte, id [32]byte, plan string, startTimeInSeconds *big.Int, tokens *big.Int) error {
	if !bytes.Equal(vcid[:], id[:]) {
		return errors.Errorf("subscription id %x differs from my virtual chain id (%x)", id, vcid)
	}

	blockTime := time.Unix(0, int64(env.GetBlockTimestamp()))
	startTime := time.Unix(startTimeInSeconds.Int64(), 0)
	if !startTime.Before(blockTime) {
		return errors.Errorf("subscription isn't valid because it only starts at %s", startTime)
	}

	if planCostInTokens, ok := plans[plan]; !ok {
		return errors.Errorf("plan with name %s is not recognized", plan)
	} else if big.NewInt(int64(planCostInTokens)).Cmp(tokens) < 0 {
		return errors.Errorf("plan with name %s costs %d tokens but subscription only paid %s tokens", plan, planCostInTokens, tokens)
	}

	return nil
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
