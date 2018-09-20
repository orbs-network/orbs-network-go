package repository

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkContract"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Info"
)

var PreBuiltContracts = map[string]*sdk.ContractInfo{
	globalpreorder_systemcontract.CONTRACT.Name: &globalpreorder_systemcontract.CONTRACT,
	deployments_systemcontract.CONTRACT.Name:    &deployments_systemcontract.CONTRACT,
	info_systemcontract.CONTRACT.Name:           &info_systemcontract.CONTRACT,
	benchmarkcontract.CONTRACT.Name:             &benchmarkcontract.CONTRACT,
	benchmarktoken.CONTRACT.Name:                &benchmarktoken.CONTRACT,
	// add new pre-built native system contracts here
}
