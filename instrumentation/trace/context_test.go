// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package trace

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"testing"
)

func TestEntryPoint_DecoratesContext(t *testing.T) {
	ctx := NewContext(context.Background(), "foo")

	ep, ok := FromContext(ctx)

	require.True(t, ok)
	require.Equal(t, "foo", ep.name)
	require.NotEmpty(t, ep.requestId)
}

func TestNestedContextsRetainValue(t *testing.T) {
	ctx := NewContext(context.Background(), "foo")
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ep, ok := FromContext(childCtx)

	require.True(t, ok)
	require.Equal(t, "foo", ep.name)
	require.NotEmpty(t, ep.requestId)
}

func TestPropagateContextRetainsValue(t *testing.T) {
	ctx := NewContext(context.Background(), "foo")
	ep, ok := FromContext(ctx)

	anotherCtx := context.Background()
	propgatedTracingContext, ok := FromContext(PropagateContext(anotherCtx, ep))

	require.True(t, ok)
	require.Equal(t, "foo", propgatedTracingContext.name)
	require.NotEmpty(t, propgatedTracingContext.requestId)
}

func TestTranslateToRequestAndBack(t *testing.T) {
	ctx := NewContext(context.Background(), "foo")
	ep, _ := FromContext(ctx)

	request, _ := http.NewRequest("Get", "localhost", nil)
	ep.WriteTraceToRequest(request)

	require.Equal(t, "foo", request.Header.Get(RequestTraceName))

	fctx := NewFromRequest(context.Background(), request)
	ep2, ok := FromContext(fctx)
	require.True(t, ok)
	require.Equal(t, ep.name, ep2.name)
	require.Equal(t, ep.requestId, ep2.requestId)
	require.True(t, ep.created.Equal(ep2.created))
}

func TestValidateRequestIdFormat(t *testing.T) {
	const nodeId = "testNodeId"
	const entryPoint = "testEntryPoint"

	ctx := ContextWithNodeId(context.Background(), nodeId)
	ctxWithTracingCtx := NewContext(ctx, entryPoint)
	tracingCtx, ok := FromContext(ctxWithTracingCtx)
	require.True(t, ok)

	requestId := tracingCtx.requestId
	fields := strings.Split(requestId, "-")
	require.Equal(t, fields[0], entryPoint, "expected entry point in request id to match")
	require.Equal(t, fields[1], nodeId, "expected node id in request id to match")
}

func TestValidateRequestIdFormatWhenNodeIdNotAvailable(t *testing.T) {
	const entryPoint = "testEntryPoint"

	ctxWithTracingCtx := NewContext(context.Background(), entryPoint)
	tracingCtx, ok := FromContext(ctxWithTracingCtx)
	require.True(t, ok)

	requestId := tracingCtx.requestId
	fields := strings.Split(requestId, "-")
	require.Equal(t, fields[0], entryPoint, "expected entry point in request id to match")
	require.Equal(t, fields[1], defaultNodeId, "expected node id in request id to match the default id")
}
