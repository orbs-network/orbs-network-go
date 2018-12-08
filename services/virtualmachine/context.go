package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"sync"
)

type executionContext struct {
	contextId           primitives.ExecutionContextId
	blockHeight         primitives.BlockHeight
	blockTimestamp      primitives.TimestampNano
	serviceStack        []primitives.ContractName
	transientState      *transientState
	accessScope         protocol.ExecutionAccessScope
	batchTransientState *transientState
	transaction         *protocol.Transaction
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

func (c *executionContext) serviceStackDepth() int {
	return len(c.serviceStack)
}

func (c *executionContext) serviceStackPeekCurrent() primitives.ContractName {
	if len(c.serviceStack) == 0 {
		return ""
	}
	return c.serviceStack[len(c.serviceStack)-1]
}

func (c *executionContext) serviceStackPeekCaller() primitives.ContractName {
	if len(c.serviceStack) <= 1 {
		return ""
	}
	return c.serviceStack[len(c.serviceStack)-2]
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

func (cp *executionContextProvider) allocateExecutionContext(blockHeight primitives.BlockHeight, blockTimestamp primitives.TimestampNano, accessScope protocol.ExecutionAccessScope, transaction *protocol.Transaction) (primitives.ExecutionContextId, *executionContext) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	newContext := &executionContext{
		blockHeight:    blockHeight,
		blockTimestamp: blockTimestamp,
		serviceStack:   []primitives.ContractName{},
		transientState: newTransientState(),
		accessScope:    accessScope,
		transaction:    transaction,
	}

	// TODO(https://github.com/orbs-network/orbs-network-go/issues/570): improve this mechanism because it wraps around on overflow
	cp.lastContextId += 1
	newContextId := cp.lastContextId
	newContext.contextId = newContextId
	cp.activeContexts[newContextId] = newContext
	return newContextId, newContext
}

func (cp *executionContextProvider) destroyExecutionContext(executionContextId primitives.ExecutionContextId) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	delete(cp.activeContexts, executionContextId)
}

func (cp *executionContextProvider) loadExecutionContext(executionContextId primitives.ExecutionContextId) *executionContext {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	return cp.activeContexts[executionContextId]
}
