// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package trace

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"time"
)

type entryPointKeyType string

const entryPointKey entryPointKeyType = "ep"
const RequestId = "request-id"

type Context struct {
	created   time.Time
	name      string
	requestId string
}

func NewContext(parent context.Context, name string) context.Context {
	now := time.Now()
	ep := &Context{
		name:      name,
		created:   now,
		requestId: fmt.Sprintf("%s-%d", name, now.UnixNano()),
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
