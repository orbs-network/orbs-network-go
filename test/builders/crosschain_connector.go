package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type ethereumConnector struct {
	timestamp       primitives.TimestampNano
	functionName    string
	abi             string
	contractAddress string
	packedArgs      []byte
}

func EthereumCallContractInput() *ethereumConnector {
	return &ethereumConnector{
		timestamp:       primitives.TimestampNano(time.Now().UnixNano()),
		functionName:    "placeholder",
		packedArgs:      nil,
		contractAddress: "0xABCDEF",
		abi:             "[]",
	}
}

func (ec *ethereumConnector) WithTimestamp(t time.Time) *ethereumConnector {
	ec.timestamp = primitives.TimestampNano(t.UnixNano())
	return ec
}

func (ec *ethereumConnector) WithFunctionName(name string) *ethereumConnector {
	ec.functionName = name
	return ec
}

func (ec *ethereumConnector) WithAbi(json string) *ethereumConnector {
	ec.abi = json
	return ec
}

func (ec *ethereumConnector) WithContractAddress(address string) *ethereumConnector {
	ec.contractAddress = address
	return ec
}

func (ec *ethereumConnector) WithPackedArguments(data []byte) *ethereumConnector {
	ec.packedArgs = data
	return ec
}

func (ec *ethereumConnector) Build() *services.EthereumCallContractInput {
	return &services.EthereumCallContractInput{
		ReferenceTimestamp:              ec.timestamp,
		EthereumAbiPackedInputArguments: ec.packedArgs,
		EthereumContractAddress:         ec.contractAddress,
		EthereumJsonAbi:                 ec.abi,
		EthereumFunctionName:            ec.functionName,
	}
}
