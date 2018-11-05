package trace

import (
	"context"
	"github.com/stretchr/testify/require"
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
