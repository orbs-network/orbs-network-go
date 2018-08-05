package repository

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkContract"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
)

var Contracts = []types.ContractInfo{
	benchmarkcontract.CONTRACT,
	benchmarktoken.CONTRACT,
	// add contracts here
}
