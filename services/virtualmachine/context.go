package virtualmachine

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type executionContext struct {
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
	return c.serviceStack[len(c.serviceStack)-1]
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

func (s *service) allocateExecutionContext(blockHeight primitives.BlockHeight, accessScope protocol.ExecutionAccessScope) (primitives.ExecutionContextId, *executionContext) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newContext := &executionContext{
		blockHeight:    blockHeight,
		serviceStack:   []primitives.ContractName{},
		transientState: newTransientState(),
		accessScope:    accessScope,
	}

	// TODO: improve this mechanism because it wraps around
	s.lastContextId += 1
	res := s.lastContextId
	s.activeContexts[res] = newContext
	return res, newContext
}

func (s *service) destroyExecutionContext(contextId primitives.ExecutionContextId) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.activeContexts, contextId)
}

func (s *service) loadExecutionContext(contextId primitives.ExecutionContextId) *executionContext {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.activeContexts[contextId]
}
