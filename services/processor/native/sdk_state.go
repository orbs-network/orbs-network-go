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
	handler handlers.ContractSdkCallHandler
}

const SDK_STATE_CONTRACT_NAME = "Sdk.State"

func (s *stateSdk) ReadBytesByAddress(ctx types.Context, address primitives.Ripmd160Sha256) ([]byte, error) {
	output, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:    primitives.ExecutionContextId(ctx),
		ContractName: SDK_STATE_CONTRACT_NAME,
		MethodName:   "read",
		InputArguments: []*protocol.MethodArgument{
			(&protocol.MethodArgumentBuilder{
				Name:       "key",
				Type:       protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE,
				BytesValue: address,
			}).Build(),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(output.OutputArguments) != 1 || !output.OutputArguments[0].IsTypeBytesValue() {
		return nil, errors.Errorf("read Sdk.State returned corrupt output value")
	}
	return output.OutputArguments[0].BytesValue(), nil
}

func (s *stateSdk) WriteBytesByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value []byte) error {
	_, err := s.handler.HandleSdkCall(&handlers.HandleSdkCallInput{
		ContextId:    primitives.ExecutionContextId(ctx),
		ContractName: SDK_STATE_CONTRACT_NAME,
		MethodName:   "write",
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
	})
	return err
}

func (s *stateSdk) ReadBytesByKey(ctx types.Context, key string) ([]byte, error) {
	address := keyToAddress(key)
	return s.ReadBytesByAddress(ctx, address)
}

func (s *stateSdk) ReadStringByAddress(ctx types.Context, address primitives.Ripmd160Sha256) (string, error) {
	bytes, err := s.ReadBytesByAddress(ctx, address)
	return string(bytes), err
}

func (s *stateSdk) ReadStringByKey(ctx types.Context, key string) (string, error) {
	address := keyToAddress(key)
	return s.ReadStringByAddress(ctx, address)
}

func (s *stateSdk) ReadUint64ByAddress(ctx types.Context, address primitives.Ripmd160Sha256) (uint64, error) {
	bytes, err := s.ReadBytesByAddress(ctx, address)
	if err != nil || len(bytes) == 0 {
		return 0, err
	}
	return membuffers.GetUint64(bytes), nil // TODO: maybe we need GetUint64Polyfill if we cannot guarantee alignment
}

func (s *stateSdk) ReadUint64ByKey(ctx types.Context, key string) (uint64, error) {
	address := keyToAddress(key)
	return s.ReadUint64ByAddress(ctx, address)
}

func (s *stateSdk) ReadUint32ByAddress(ctx types.Context, address primitives.Ripmd160Sha256) (uint32, error) {
	bytes, err := s.ReadBytesByAddress(ctx, address)
	if err != nil || len(bytes) == 0 {
		return 0, nil
	}
	return membuffers.GetUint32(bytes), nil // TODO: maybe we need GetUint32Polyfill if we cannot guarantee alignment
}

func (s *stateSdk) ReadUint32ByKey(ctx types.Context, key string) (uint32, error) {
	address := keyToAddress(key)
	return s.ReadUint32ByAddress(ctx, address)
}

func (s *stateSdk) WriteBytesByKey(ctx types.Context, key string, value []byte) error {
	address := keyToAddress(key)
	return s.WriteBytesByAddress(ctx, address, value)
}

func (s *stateSdk) WriteStringByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value string) error {
	bytes := []byte(value)
	return s.WriteBytesByAddress(ctx, address, bytes)
}

func (s *stateSdk) WriteStringByKey(ctx types.Context, key string, value string) error {
	address := keyToAddress(key)
	return s.WriteStringByAddress(ctx, address, value)
}

func (s *stateSdk) WriteUint64ByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value uint64) error {
	bytes := make([]byte, 8)
	membuffers.WriteUint64(bytes, value) // TODO: maybe we need WriteUint64Polyfill if we cannot guarantee alignment
	return s.WriteBytesByAddress(ctx, address, bytes)
}

func (s *stateSdk) WriteUint64ByKey(ctx types.Context, key string, value uint64) error {
	address := keyToAddress(key)
	return s.WriteUint64ByAddress(ctx, address, value)
}

func (s *stateSdk) WriteUint32ByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value uint32) error {
	bytes := make([]byte, 4)
	membuffers.WriteUint32(bytes, value) // TODO: maybe we need WriteUint32Polyfill if we cannot guarantee alignment
	return s.WriteBytesByAddress(ctx, address, bytes)
}

func (s *stateSdk) WriteUint32ByKey(ctx types.Context, key string, value uint32) error {
	address := keyToAddress(key)
	return s.WriteUint32ByAddress(ctx, address, value)
}

func (s *stateSdk) ClearByAddress(ctx types.Context, address primitives.Ripmd160Sha256) error {
	return s.WriteBytesByAddress(ctx, address, []byte{})
}

func (s *stateSdk) ClearByKey(ctx types.Context, key string) error {
	address := keyToAddress(key)
	return s.ClearByAddress(ctx, address)
}

func keyToAddress(key string) primitives.Ripmd160Sha256 {
	return hash.CalcRipmd160Sha256([]byte(key))
}
