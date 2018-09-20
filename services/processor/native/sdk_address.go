package native

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

type addressSdk struct {
	handler         handlers.ContractSdkCallHandler
	permissionScope protocol.ExecutionPermissionScope
}

const SDK_OPERATION_NAME_ADDRESS = "Sdk.Address"

func (s *addressSdk) ValidateAddress(executionContextId sdk.Context, address sdk.Ripmd160Sha256) error {
	if len(address) != hash.RIPMD160_HASH_SIZE_BYTES {
		return errors.Errorf("valid address length is %d bytes, received %d bytes", hash.RIPMD160_HASH_SIZE_BYTES, len(address))
	}
	return nil
}

func (s *addressSdk) GetSignerAddress(executionContextId sdk.Context) (sdk.Ripmd160Sha256, error) {
	output, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ADDRESS,
		MethodName:      "getSignerAddress",
		InputArguments:  []*protocol.MethodArgument{},
		PermissionScope: s.permissionScope,
	})
	if err != nil {
		return nil, err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return nil, errors.Errorf("getSignerAddress Sdk.Address returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue(), nil
}

func (s *addressSdk) GetCallerAddress(executionContextId sdk.Context) (sdk.Ripmd160Sha256, error) {
	output, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:       primitives.ExecutionContextId(executionContextId),
		OperationName:   SDK_OPERATION_NAME_ADDRESS,
		MethodName:      "getCallerAddress",
		InputArguments:  []*protocol.MethodArgument{},
		PermissionScope: s.permissionScope,
	})
	if err != nil {
		return nil, err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return nil, errors.Errorf("getCallerAddress Sdk.Address returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue(), nil
}
