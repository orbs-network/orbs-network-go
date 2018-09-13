package repository

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkContract"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
)

var PreBuiltContracts = map[string]sdk.ContractInfo{
	globalpreorder.CONTRACT.Name:    globalpreorder.CONTRACT,
	deployments.CONTRACT.Name:       deployments.CONTRACT,
	benchmarkcontract.CONTRACT.Name: benchmarkcontract.CONTRACT,
	benchmarktoken.CONTRACT.Name:    benchmarktoken.CONTRACT,
	// add new native system contracts here
}
