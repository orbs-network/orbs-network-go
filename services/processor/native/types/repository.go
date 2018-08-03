package types

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type ContractInfo struct {
	Name           primitives.ContractName
	Permission     protocol.ExecutionPermissionScope
	Methods        []MethodInfo
	Implementation func(*BaseContract) Contract
}

type MethodInfo struct {
	Name           primitives.MethodName
	External       bool
	Access         protocol.ExecutionAccessScope
	Implementation interface{}
}
