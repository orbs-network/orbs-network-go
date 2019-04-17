// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package logfields

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContextStringValue_NotInContext(t *testing.T) {
	ctx := context.Background()
	f := ContextStringValue(ctx, "a")
	require.Equal(t, "not-found-in-context", f.StringVal, "expected to have the default not found value")
}

func TestContextStringValue_InContextNotAString(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "a", 14)
	f := ContextStringValue(ctx, "a")
	require.Equal(t, "found-in-context-but-not-string", f.StringVal, "expected to have the default not a string value")
}

func TestContextStringValue_InContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "a", "found")
	f := ContextStringValue(ctx, "a")
	require.Equal(t, "found", f.StringVal, "expected to be part of the context")
}
