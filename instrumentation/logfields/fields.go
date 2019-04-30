// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package logfields

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
)

func Transaction(txHash primitives.Sha256) *log.Field {
	return log.Stringable("txHash", txHash)
}

func Query(queryHash primitives.Sha256) *log.Field {
	return log.Stringable("queryHash", queryHash)
}

func TimestampNano(key string, value primitives.TimestampNano) *log.Field {
	return &log.Field{Key: key, Int: int64(value), Type: log.TimeType}
}

func BlockHeight(value primitives.BlockHeight) *log.Field {
	return &log.Field{Key: "block-height", Uint: uint64(value), Type: log.UintType}
}

func VirtualChainId(value primitives.VirtualChainId) *log.Field {
	return &log.Field{Key: "vcid", Uint: uint64(value), Type: log.UintType}
}

func ContextStringValue(ctx context.Context, key string) *log.Field {
	val := "not-found-in-context"
	if v := ctx.Value(key); v != nil {
		if vString, ok := v.(string); ok {
			val = vString
		} else {
			val = "found-in-context-but-not-string"
		}
	}
	return log.String(key, val)
}
