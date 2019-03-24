// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package globalpreorder_systemcontract

import (
	"encoding/binary"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/env"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/ethereum"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
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

var satoshiFactor = big.NewInt(1000000000000000000)

var planCostsInOrbs = map[string]int64{
	"B0": 450,
	"B1": 850,
	"B2": 1650,
	"B3": 3300,
	"B4": 6600,
	"B5": 13200,
}

type subscriptionData struct {
	id               primitives.VirtualChainId
	plan             string
	startTime        time.Time
	tokensPaidInOrbs int64
}

func (s *subscriptionData) validate(virtualChainId primitives.VirtualChainId, blockTime time.Time) error {
	if s.id != virtualChainId {
		return errors.Errorf("subscription id %d differs from my virtual chain id (%d)", s.id, virtualChainId)
	}

	if !s.startTime.Before(blockTime) {
		return errors.Errorf("subscription isn't valid because it only starts at %s", s.startTime)
	}

	if planCostInTokens, ok := planCostsInOrbs[s.plan]; !ok {
		return errors.Errorf("plan with name %s is not recognized", s.plan)
	} else if s.tokensPaidInOrbs < planCostInTokens {
		return errors.Errorf("plan with name %s costs %d tokens but subscription only paid %d tokens", s.plan, planCostInTokens, s.tokensPaidInOrbs)
	}

	return nil
}

func refreshSubscription(ethContractAddress string) {
	currentContract := _readSubscriptionContract()

	if len(currentContract) != 0 && currentContract != ethContractAddress {
		panic(fmt.Sprintf("can only refresh current contract %s", currentContract))
	}

	if len(currentContract) == 0 {
		_writeSubscriptionContract(ethContractAddress)
	}

	myVirtualChainId := primitives.VirtualChainId(env.GetVirtualChainId())

	subscription := _readSubscriptionDataFromEthereum(myVirtualChainId, ethContractAddress)
	if err := subscription.validate(myVirtualChainId, time.Unix(0, int64(env.GetBlockTimestamp()))); err == nil {
		_clearSubscriptionProblem()
	} else {
		_writeSubscriptionProblem(err.Error())
	}
}

func _readSubscriptionDataFromEthereum(virtualChainId primitives.VirtualChainId, ethContractAddress string) *subscriptionData {
	jsonAbi := `[{"constant":true,"inputs":[{"name":"_id","type":"bytes32"}],"name":"getSubscriptionData","outputs":[{"name":"id","type":"bytes32"},{"name":"profile","type":"string"},{"name":"startTime","type":"uint256"},{"name":"tokens","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`
	plan := ""
	id := [32]byte{}
	startTime := big.NewInt(0)
	tokens := big.NewInt(0)

	var output [4]interface{}
	output[0] = &id
	output[1] = &plan
	output[2] = &startTime
	output[3] = &tokens

	var vcid [32]byte
	binary.BigEndian.PutUint32(vcid[28:], uint32(virtualChainId))
	ethereum.CallMethod(ethContractAddress, jsonAbi, "getSubscriptionData", &output, vcid)
	subscription := &subscriptionData{
		id:               primitives.VirtualChainId(binary.BigEndian.Uint32(id[28:])),
		plan:             plan,
		startTime:        time.Unix(startTime.Int64(), 0),
		tokensPaidInOrbs: _satoshiToOrbs(tokens),
	}
	return subscription
}

func _satoshiToOrbs(tokens *big.Int) int64 {
	return tokens.Div(tokens, satoshiFactor).Int64()
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
