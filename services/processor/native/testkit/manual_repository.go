package testkit

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"sync"
)

type ManualRepository struct {
	sync.Mutex
	contracts map[string]*sdkContext.ContractInfo
}

func NewRepository() *ManualRepository {
	return &ManualRepository{contracts: make(map[string]*sdkContext.ContractInfo)}
}

func (r *ManualRepository) ContractInfo(ctx context.Context, executionContextId primitives.ExecutionContextId, contractName string) (*sdkContext.ContractInfo, error) {
	r.Lock()
	defer r.Unlock()
	return r.contracts[contractName], nil
}

func (r *ManualRepository) Register(contractName string, publicMethods []interface{}, systemMethods []interface{}, events []interface{}, permissions sdkContext.PermissionScope) {
	r.Lock()
	defer r.Unlock()
	r.contracts[contractName] = &sdkContext.ContractInfo{PublicMethods: publicMethods, SystemMethods: systemMethods, EventsMethods: events, Permission: permissions}
}
