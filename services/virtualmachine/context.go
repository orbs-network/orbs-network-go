package virtualmachine

import "github.com/orbs-network/orbs-spec/types/go/primitives"

type executionContext struct {
	blockHeight    primitives.BlockHeight
	serviceStack   []primitives.ContractName
	transientState *transientState
}

func (c *executionContext) serviceStackTop() primitives.ContractName {
	if len(c.serviceStack) == 0 {
		return ""
	}
	return c.serviceStack[len(c.serviceStack)-1]
}

func (s *service) allocateExecutionContext(blockHeight primitives.BlockHeight, callingService primitives.ContractName, withTransientState bool) (res primitives.ExecutionContextId) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var transientState *transientState
	if withTransientState {
		transientState = newTransientState()
	}

	newContext := &executionContext{
		blockHeight:    blockHeight,
		serviceStack:   []primitives.ContractName{callingService},
		transientState: transientState,
	}

	// TODO: improve this mechanism because it wraps around
	s.lastContextId += 1
	res = s.lastContextId
	s.activeContexts[res] = newContext
	return
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
