package native

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
)

type stateSdk struct {
	handler handlers.ContractSdkCallHandler
}

func (s *stateSdk) ReadBytesByAddress(ctx types.Context, address primitives.Ripmd160Sha256) []byte {
	panic("Not implemented")
}

func (s *stateSdk) ReadBytesByKey(ctx types.Context, key string) []byte {
	panic("Not implemented")
}

func (s *stateSdk) ReadStringByAddress(ctx types.Context, address primitives.Ripmd160Sha256) string {
	panic("Not implemented")
}

func (s *stateSdk) ReadStringByKey(ctx types.Context, key string) string {
	panic("Not implemented")
}

func (s *stateSdk) ReadUint64ByAddress(ctx types.Context, address primitives.Ripmd160Sha256) uint64 {
	panic("Not implemented")
}

func (s *stateSdk) ReadUint64ByKey(ctx types.Context, key string) uint64 {
	return 17 // TODO: temp
}

func (s *stateSdk) ReadUint32ByAddress(ctx types.Context, address primitives.Ripmd160Sha256) uint32 {
	panic("Not implemented")
}

func (s *stateSdk) ReadUint32ByKey(ctx types.Context, key string) uint32 {
	panic("Not implemented")
}

func (s *stateSdk) WriteBytesByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value []byte) error {
	panic("Not implemented")
}

func (s *stateSdk) WriteBytesByKey(ctx types.Context, key string, value []byte) error {
	panic("Not implemented")
}

func (s *stateSdk) WriteStringByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value string) error {
	panic("Not implemented")
}

func (s *stateSdk) WriteStringByKey(ctx types.Context, key string, value string) error {
	panic("Not implemented")
}

func (s *stateSdk) WriteUint64ByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value uint64) error {
	panic("Not implemented")
}

func (s *stateSdk) WriteUint64ByKey(ctx types.Context, key string, value uint64) error {
	return nil // TODO: temp
}

func (s *stateSdk) WriteUint32ByAddress(ctx types.Context, address primitives.Ripmd160Sha256, value uint32) error {
	panic("Not implemented")
}

func (s *stateSdk) WriteUint32ByKey(ctx types.Context, key string, value uint32) error {
	panic("Not implemented")
}

func (s *stateSdk) ClearByAddress(ctx types.Context, address primitives.Ripmd160Sha256) error {
	panic("Not implemented")
}

func (s *stateSdk) ClearByKey(ctx types.Context, key string) error {
	panic("Not implemented")
}
