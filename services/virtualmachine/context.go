package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

type executionContext struct {
	contextId           primitives.ExecutionContextId
	blockHeight         primitives.BlockHeight
	serviceStack        []primitives.ContractName
	transientState      *transientState
	accessScope         protocol.ExecutionAccessScope
	batchTransientState *transientState
}

func (c *executionContext) serviceStackTop() primitives.ContractName {
	if len(c.serviceStack) == 0 {
		return ""
	}
	res := c.serviceStack[len(c.serviceStack)-1]
	return res
}

func (c *executionContext) serviceStackPush(service primitives.ContractName) {
	c.serviceStack = append(c.serviceStack, service)
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
		serviceStack:   []primitives.ContractName{},
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
