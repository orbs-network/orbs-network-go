package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

type serviceAndPermission struct {
	service    primitives.ContractName
	permission protocol.ExecutionPermissionScope
}

type executionContext struct {
	contextId           primitives.ExecutionContextId
	blockHeight         primitives.BlockHeight
	serviceStack        []*serviceAndPermission
	transientState      *transientState
	accessScope         protocol.ExecutionAccessScope
	batchTransientState *transientState
}

func (c *executionContext) serviceStackTop() (primitives.ContractName, protocol.ExecutionPermissionScope) {
	if len(c.serviceStack) == 0 {
		return "", 0
	}
	res := c.serviceStack[len(c.serviceStack)-1]
	return res.service, res.permission
}

func (c *executionContext) serviceStackPush(service primitives.ContractName, servicePermission protocol.ExecutionPermissionScope) {
	c.serviceStack = append(c.serviceStack, &serviceAndPermission{service, servicePermission})
}

func (c *executionContext) serviceStackPop() {
	if len(c.serviceStack) == 0 {
		return
	}
	c.serviceStack = c.serviceStack[0 : len(c.serviceStack)-1]
}

type executionContextProvider struct {
	mutex          *sync.RWMutex
	activeContexts map[primitives.ExecutionContextId]*executionContext
	lastContextId  primitives.ExecutionContextId
}

func newExecutionContextProvider() *executionContextProvider {
	return &executionContextProvider{
		mutex:          &sync.RWMutex{},
		activeContexts: make(map[primitives.ExecutionContextId]*executionContext),
	}
}

func (cp *executionContextProvider) allocateExecutionContext(blockHeight primitives.BlockHeight, accessScope protocol.ExecutionAccessScope) (primitives.ExecutionContextId, *executionContext) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	newContext := &executionContext{
		blockHeight:    blockHeight,
		serviceStack:   []*serviceAndPermission{},
		transientState: newTransientState(),
		accessScope:    accessScope,
	}

	// TODO: improve this mechanism because it wraps around on overflow
	cp.lastContextId += 1
	newContextId := cp.lastContextId
	newContext.contextId = newContextId
	cp.activeContexts[newContextId] = newContext
	return newContextId, newContext
}

func (cp *executionContextProvider) destroyExecutionContext(contextId primitives.ExecutionContextId) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	delete(cp.activeContexts, contextId)
}

func (cp *executionContextProvider) loadExecutionContext(contextId primitives.ExecutionContextId) *executionContext {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	return cp.activeContexts[contextId]
}
