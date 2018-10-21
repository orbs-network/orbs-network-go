package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContext_Load(t *testing.T) {
	cp := newExecutionContextProvider()

	contextId1, _ := cp.allocateExecutionContext(1, protocol.ACCESS_SCOPE_READ_ONLY, nil)
	defer cp.destroyExecutionContext(contextId1)

	contextId2, _ := cp.allocateExecutionContext(2, protocol.ACCESS_SCOPE_READ_ONLY, nil)
	defer cp.destroyExecutionContext(contextId1)

	require.NotEqual(t, contextId1, contextId2, "contextId1 should be different from contextId2")

	c1 := cp.loadExecutionContext(contextId1)
	require.EqualValues(t, 1, c1.blockHeight, "loaded context with contextId1 should be 1")

	c2 := cp.loadExecutionContext(contextId2)
	require.EqualValues(t, 2, c2.blockHeight, "loaded context with contextId2 should be 2")
}

func TestContext_ServiceStack(t *testing.T) {
	cp := newExecutionContextProvider()
	executionContextId, c := cp.allocateExecutionContext(1, protocol.ACCESS_SCOPE_READ_ONLY, nil)
	defer cp.destroyExecutionContext(executionContextId)

	c.serviceStackPush("Service1")
	service := c.serviceStackTop()
	require.EqualValues(t, "Service1", service, "service top should be initialized")
	require.Equal(t, 1, c.serviceStackDepth(), "service stack depth should match")
	require.EqualValues(t, "Service1", c.serviceStackPeekCurrent(), "current service should match")
	require.Zero(t, c.serviceStackPeekCaller(), "calling service should be empty")

	c.serviceStackPush("Service2")
	service = c.serviceStackTop()
	require.EqualValues(t, "Service2", service, "service top should change after push")
	require.Equal(t, 2, c.serviceStackDepth(), "service stack depth should match")
	require.EqualValues(t, "Service2", c.serviceStackPeekCurrent(), "current service should match")
	require.EqualValues(t, "Service1", c.serviceStackPeekCaller(), "calling service should match")

	c.serviceStackPop()
	service = c.serviceStackTop()
	require.EqualValues(t, "Service1", service, "service top should return to origin after pop")
	require.Equal(t, 1, c.serviceStackDepth(), "service stack depth should match")
	require.EqualValues(t, "Service1", c.serviceStackPeekCurrent(), "current service should match")
	require.Zero(t, c.serviceStackPeekCaller(), "calling service should be empty")
}
