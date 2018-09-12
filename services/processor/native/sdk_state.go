package native

import (
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/pkg/errors"
)

type stateSdk struct {
	handler         handlers.ContractSdkCallHandler
	permissionScope protocol.ExecutionPermissionScope
}

const SDK_OPERATION_NAME_STATE = "Sdk.State"

func (s *stateSdk) ReadBytesByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256) ([]byte, error) {
	output, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_STATE,
		MethodName:    "read",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:       "key",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: address,
			}).Build(),
		},
		PermissionScope: s.permissionScope,
	})
	if err != nil {
		return nil, err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return nil, errors.Errorf("read Sdk.State returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue(), nil
}

func (s *stateSdk) WriteBytesByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256, value []byte) error {
	_, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:     primitives.ExecutionContextId(executionContextId),
		OperationName: SDK_OPERATION_NAME_STATE,
		MethodName:    "write",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:       "key",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: address,
			}).Build(),
			(&protocol.MethodArgumentBuilder{
				Name:       "value",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: value,
			}).Build(),
		},
		PermissionScope: s.permissionScope,
	})
	return err
}

func (s *stateSdk) ReadBytesByKey(executionContextId types.Context, key string) ([]byte, error) {
	address := keyToAddress(key)
	return s.ReadBytesByAddress(executionContextId, address)
}

func (s *stateSdk) ReadStringByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256) (string, error) {
	bytes, err := s.ReadBytesByAddress(executionContextId, address)
	return string(bytes), err
}

func (s *stateSdk) ReadStringByKey(executionContextId types.Context, key string) (string, error) {
	address := keyToAddress(key)
	return s.ReadStringByAddress(executionContextId, address)
}

func (s *stateSdk) ReadUint64ByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256) (uint64, error) {
	bytes, err := s.ReadBytesByAddress(executionContextId, address)
	if err != nil || len(bytes) == 0 {
		return 0, err
	}
	return membuffers.GetUint64(bytes), nil // TODO: maybe we need GetUint64Polyfill if we cannot guarantee alignment
}

func (s *stateSdk) ReadUint64ByKey(executionContextId types.Context, key string) (uint64, error) {
	address := keyToAddress(key)
	return s.ReadUint64ByAddress(executionContextId, address)
}

func (s *stateSdk) ReadUint32ByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256) (uint32, error) {
	bytes, err := s.ReadBytesByAddress(executionContextId, address)
	if err != nil || len(bytes) == 0 {
		return 0, nil
	}
	return membuffers.GetUint32(bytes), nil // TODO: maybe we need GetUint32Polyfill if we cannot guarantee alignment
}

func (s *stateSdk) ReadUint32ByKey(executionContextId types.Context, key string) (uint32, error) {
	address := keyToAddress(key)
	return s.ReadUint32ByAddress(executionContextId, address)
}

func (s *stateSdk) WriteBytesByKey(executionContextId types.Context, key string, value []byte) error {
	address := keyToAddress(key)
	return s.WriteBytesByAddress(executionContextId, address, value)
}

func (s *stateSdk) WriteStringByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256, value string) error {
	bytes := []byte(value)
	return s.WriteBytesByAddress(executionContextId, address, bytes)
}

func (s *stateSdk) WriteStringByKey(executionContextId types.Context, key string, value string) error {
	address := keyToAddress(key)
	return s.WriteStringByAddress(executionContextId, address, value)
}

func (s *stateSdk) WriteUint64ByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256, value uint64) error {
	bytes := make([]byte, 8)
	membuffers.WriteUint64(bytes, value) // TODO: maybe we need WriteUint64Polyfill if we cannot guarantee alignment
	return s.WriteBytesByAddress(executionContextId, address, bytes)
}

func (s *stateSdk) WriteUint64ByKey(executionContextId types.Context, key string, value uint64) error {
	address := keyToAddress(key)
	return s.WriteUint64ByAddress(executionContextId, address, value)
}

func (s *stateSdk) WriteUint32ByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256, value uint32) error {
	bytes := make([]byte, 4)
	membuffers.WriteUint32(bytes, value) // TODO: maybe we need WriteUint32Polyfill if we cannot guarantee alignment
	return s.WriteBytesByAddress(executionContextId, address, bytes)
}

func (s *stateSdk) WriteUint32ByKey(executionContextId types.Context, key string, value uint32) error {
	address := keyToAddress(key)
	return s.WriteUint32ByAddress(executionContextId, address, value)
}

func (s *stateSdk) ClearByAddress(executionContextId types.Context, address primitives.Ripmd160Sha256) error {
	return s.WriteBytesByAddress(executionContextId, address, []byte{})
}

func (s *stateSdk) ClearByKey(executionContextId types.Context, key string) error {
	address := keyToAddress(key)
	return s.ClearByAddress(executionContextId, address)
}

func keyToAddress(key string) primitives.Ripmd160Sha256 {
	return hash.CalcRipmd160Sha256([]byte(key))
}
