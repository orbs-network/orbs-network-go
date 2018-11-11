package native

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type ethereumSdk struct {
	handler         handlers.ContractSdkCallHandler
	permissionScope protocol.ExecutionPermissionScope
}

const SDK_OPERATION_NAME_ETHEREUM = "Sdk.Ethereum"

func (s *ethereumSdk) CallMethod(executionContextId sdk.Context, contractAddress string, jsonAbi string, methodName string, out interface{}, args ...interface{}) error {
	panic("Not implemented")
}
