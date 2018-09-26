package adapter

import (
	"context"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-network-go/services/processor/native/adapter"
	"github.com/pkg/errors"
	"sync"
)

type FakeCompiler interface {
	adapter.Compiler
	ProvideFakeContract(fakeContractInfo *sdk.ContractInfo, code string)
}

type fakeCompiler struct {
	mutex    *sync.RWMutex
	provided map[string]*sdk.ContractInfo
}

func NewFakeCompiler() FakeCompiler {
	return &fakeCompiler{
		mutex:    &sync.RWMutex{},
		provided: make(map[string]*sdk.ContractInfo),
	}
}

func (c *fakeCompiler) ProvideFakeContract(fakeContractInfo *sdk.ContractInfo, code string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.provided[code] = fakeContractInfo
}

func (c *fakeCompiler) Compile(ctx context.Context, code string) (*sdk.ContractInfo, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	contractInfo, found := c.provided[code]
	if !found {
		return nil, errors.New("fake contract for given code was not previously provided with ProvideFakeContract()")
	}

	return contractInfo, nil
}
