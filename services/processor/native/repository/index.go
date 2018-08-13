package repository

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkContract"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
)

var Contracts = []types.ContractInfo{
	globalpreorder.CONTRACT,
	deployments.CONTRACT,
	benchmarkcontract.CONTRACT,
	benchmarktoken.CONTRACT,
	// add new native system contracts here
}
