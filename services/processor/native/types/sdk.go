package types

import "github.com/orbs-network/orbs-spec/types/go/primitives"

type StateSdk interface {
	// read
	ReadBytesByAddress(ctx Context, address primitives.Ripmd160Sha256) ([]byte, error)
	ReadBytesByKey(ctx Context, key string) ([]byte, error)
	ReadStringByAddress(ctx Context, address primitives.Ripmd160Sha256) (string, error)
	ReadStringByKey(ctx Context, key string) (string, error)
	ReadUint64ByAddress(ctx Context, address primitives.Ripmd160Sha256) (uint64, error)
	ReadUint64ByKey(ctx Context, key string) (uint64, error)
	ReadUint32ByAddress(ctx Context, address primitives.Ripmd160Sha256) (uint32, error)
	ReadUint32ByKey(ctx Context, key string) (uint32, error)

	// write
	WriteBytesByAddress(ctx Context, address primitives.Ripmd160Sha256, value []byte) error
	WriteBytesByKey(ctx Context, key string, value []byte) error
	WriteStringByAddress(ctx Context, address primitives.Ripmd160Sha256, value string) error
	WriteStringByKey(ctx Context, key string, value string) error
	WriteUint64ByAddress(ctx Context, address primitives.Ripmd160Sha256, value uint64) error
	WriteUint64ByKey(ctx Context, key string, value uint64) error
	WriteUint32ByAddress(ctx Context, address primitives.Ripmd160Sha256, value uint32) error
	WriteUint32ByKey(ctx Context, key string, value uint32) error

	// clear
	ClearByAddress(ctx Context, address primitives.Ripmd160Sha256) error
	ClearByKey(ctx Context, key string) error
}

type ServiceSdk interface {
	IsNative(ctx Context, serviceName string) error
	CallMethod(ctx Context, serviceName string, methodName string) error // TODO: handle var args and return
}
