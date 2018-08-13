package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestContextLoad(t *testing.T) {
	cp := newExecutionContextProvider()

	contextId1, _ := cp.allocateExecutionContext(1, protocol.ACCESS_SCOPE_READ_ONLY)
	defer cp.destroyExecutionContext(contextId1)

	contextId2, _ := cp.allocateExecutionContext(2, protocol.ACCESS_SCOPE_READ_ONLY)
	defer cp.destroyExecutionContext(contextId1)

	require.NotEqual(t, contextId1, contextId2, "contextId1 should be different from contextId2")

	c1 := cp.loadExecutionContext(contextId1)
	require.EqualValues(t, 1, c1.blockHeight, "loaded context with contextId1 should be 1")

	c2 := cp.loadExecutionContext(contextId2)
	require.EqualValues(t, 2, c2.blockHeight, "loaded context with contextId2 should be 2")
}

func TestContextServiceStack(t *testing.T) {
	cp := newExecutionContextProvider()
	contextId, c := cp.allocateExecutionContext(1, protocol.ACCESS_SCOPE_READ_ONLY)
	defer cp.destroyExecutionContext(contextId)

	c.serviceStackPush("Service1", protocol.PERMISSION_SCOPE_SYSTEM)
	service, permission := c.serviceStackTop()
	require.EqualValues(t, "Service1", service, "service top should be initialized")
	require.EqualValues(t, protocol.PERMISSION_SCOPE_SYSTEM, permission, "service top should be initialized")

	c.serviceStackPush("Service2", protocol.PERMISSION_SCOPE_SERVICE)
	service, permission = c.serviceStackTop()
	require.EqualValues(t, "Service2", service, "service top should change after push")
	require.EqualValues(t, protocol.PERMISSION_SCOPE_SERVICE, permission, "service top should change after push")

	c.serviceStackPop()
	service, permission = c.serviceStackTop()
	require.EqualValues(t, "Service1", service, "service top should return to origin after pop")
	require.EqualValues(t, protocol.PERMISSION_SCOPE_SYSTEM, permission, "service top should return to origin after pop")
}
