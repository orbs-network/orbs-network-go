package types

import "github.com/orbs-network/orbs-spec/types/go/primitives"

type StateSdk interface {
	// read
	ReadBytesByAddress(ctx Context, address primitives.Ripmd160Sha256) []byte
	ReadBytesByKey(ctx Context, key string) []byte
	ReadStringByAddress(ctx Context, address primitives.Ripmd160Sha256) string
	ReadStringByKey(ctx Context, key string) string
	ReadUint64ByAddress(ctx Context, address primitives.Ripmd160Sha256) uint64
	ReadUint64ByKey(ctx Context, key string) uint64
	ReadUint32ByAddress(ctx Context, address primitives.Ripmd160Sha256) uint32
	ReadUint32ByKey(ctx Context, key string) uint32

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
