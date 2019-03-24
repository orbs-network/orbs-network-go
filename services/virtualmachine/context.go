// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package virtualmachine

import (
	"crypto/rand"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"math/big"
	"sync"
)

const EXECUTION_CONTEXT_ID_SIZE_BYTES = hash.SHA256_HASH_SIZE_BYTES

var BIG_INT_ONE = big.NewInt(1)
var EXECUTION_CONTEXT_ID_BIGINT_RANGE = new(big.Int).Lsh(BIG_INT_ONE, 8*EXECUTION_CONTEXT_ID_SIZE_BYTES)

type executionContext struct {
	contextId                primitives.ExecutionContextId
	lastCommittedBlockHeight primitives.BlockHeight
	currentBlockHeight       primitives.BlockHeight
	currentBlockTimestamp    primitives.TimestampNano
	serviceStack             []primitives.ContractName
	transientState           *transientState
	accessScope              protocol.ExecutionAccessScope
	batchTransientState      *transientState
	transactionOrQuery       TransactionOrQuery
	eventList                []*protocol.EventBuilder
}

func (c *executionContext) serviceStackTop() primitives.ContractName {
	if len(c.serviceStack) == 0 {
		return ""
	}
	res := c.serviceStack[len(c.serviceStack)-1]
	return res
}

// TODO(micro-services): this will need to become thread-safe once we switch to micro-services, now only one goroutine accesses the executionContext
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

func (c *executionContext) eventListAdd(eventName primitives.EventName, opaqueArgumentArray []byte) {
	event := &protocol.EventBuilder{
		ContractName:        c.serviceStackPeekCurrent(),
		EventName:           eventName,
		OutputArgumentArray: opaqueArgumentArray,
	}
	c.eventList = append(c.eventList, event)
}

type executionContextProvider struct {
	mutex                *sync.RWMutex
	activeContexts       map[string]*executionContext
	lastContextIdCounter *big.Int
	contextIdSalt        *big.Int
}

func newExecutionContextProvider() *executionContextProvider {
	salt, err := rand.Int(rand.Reader, EXECUTION_CONTEXT_ID_BIGINT_RANGE)
	if err != nil {
		panic(err)
	}
	return &executionContextProvider{
		mutex:                &sync.RWMutex{},
		activeContexts:       make(map[string]*executionContext),
		lastContextIdCounter: big.NewInt(0),
		contextIdSalt:        salt,
	}
}

func (cp *executionContextProvider) allocateExecutionContext(lastCommittedBlockHeight primitives.BlockHeight, currentBlockHeight primitives.BlockHeight, currentBlockTimestamp primitives.TimestampNano, accessScope protocol.ExecutionAccessScope, transactionOrQuery TransactionOrQuery) (primitives.ExecutionContextId, *executionContext) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	newContext := &executionContext{
		lastCommittedBlockHeight: lastCommittedBlockHeight,
		currentBlockHeight:       currentBlockHeight,
		currentBlockTimestamp:    currentBlockTimestamp,
		serviceStack:             []primitives.ContractName{},
		transientState:           newTransientState(),
		accessScope:              accessScope,
		transactionOrQuery:       transactionOrQuery,
	}

	cp.lastContextIdCounter.Add(cp.lastContextIdCounter, BIG_INT_ONE)
	newContextId := primitives.ExecutionContextId(hash.CalcSha256(cp.contextIdSalt.Bytes(), cp.lastContextIdCounter.Bytes()))
	newContext.contextId = newContextId
	cp.activeContexts[string(newContextId)] = newContext
	return newContextId, newContext
}

func (cp *executionContextProvider) destroyExecutionContext(executionContextId primitives.ExecutionContextId) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	delete(cp.activeContexts, string(executionContextId))
}

func (cp *executionContextProvider) loadExecutionContext(executionContextId primitives.ExecutionContextId) *executionContext {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	return cp.activeContexts[string(executionContextId)]
}
