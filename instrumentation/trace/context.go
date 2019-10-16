// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package trace

import (
	"context"
	"fmt"
	"github.com/orbs-network/scribe/log"
	"net/http"
	"time"
)

type entryPointKeyType string

const entryPointKey entryPointKeyType = "ep"
const RequestId = "request-id"
const NodeIdCtxKey = "node-id"

const defaultNodeId = "none"

type Context struct {
	created   time.Time
	name      string
	requestId string
}

const RequestTraceName = "X-ORBS-NAME"
const RequestTraceTime = "X-ORBS-CREATED"
const RequestTraceRequestId = "X-ORBS-ID"

func ContextWithNodeId(ctx context.Context, nodeId string) context.Context {
	return context.WithValue(ctx, NodeIdCtxKey, nodeId)
}

func NewFromRequest(ctx context.Context, request *http.Request) context.Context {
	name := request.Header.Get(RequestTraceName)
	if name == "" {
		return ctx
	}

	created := time.Now()
	if n, err := time.Parse(time.RFC3339Nano, request.Header.Get(RequestTraceTime)); err == nil {
		created = n
	}

	traceContext := &Context{
		name:      name,
		created:   created,
		requestId: request.Header.Get(RequestTraceRequestId),
	}
	return PropagateContext(ctx, traceContext)
}

func (c *Context) WriteTraceToRequest(request *http.Request) {
	request.Header.Set(RequestTraceName, c.name)
	request.Header.Set(RequestTraceTime, c.created.Format(time.RFC3339Nano))
	request.Header.Set(RequestTraceRequestId, c.requestId)
}

func NewContext(parent context.Context, name string) context.Context {
	now := time.Now()
	nodeId := parent.Value(NodeIdCtxKey)
	if nodeId == nil {
		nodeId = defaultNodeId
	}
	ep := &Context{
		name:      name,
		created:   now,
		requestId: fmt.Sprintf("%s-%s-%d", name, nodeId, now.UnixNano()),
	}
	return context.WithValue(parent, entryPointKey, ep)
}

func PropagateContext(parent context.Context, tracingContext *Context) context.Context {
	return context.WithValue(parent, entryPointKey, tracingContext)
}

func FromContext(ctx context.Context) (e *Context, ok bool) {
	e, ok = ctx.Value(entryPointKey).(*Context)
	return
}

func (c *Context) NestedFields() []*log.Field {
	if c == nil { // this can happen if the tracing.Context was never created, e.g. context logged doesn't have this context value
		return nil
	}

	return []*log.Field{
		log.String("entry-point", c.name),
		log.String(RequestId, c.requestId),
	}
}

func LogFieldFrom(ctx context.Context) *log.Field {
	if trace, ok := FromContext(ctx); ok {
		return &log.Field{Key: "trace", Nested: trace, Type: log.AggregateType}
	} else {
		return log.String("trace", "NO-CONTEXT")
	}

}
