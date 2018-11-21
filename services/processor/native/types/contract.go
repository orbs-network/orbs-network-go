package types

import (
	"github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/pkg/errors"
	"reflect"
	"runtime"
	"strings"
)

type MethodInstance interface{}

type ContractInstance struct {
	PublicMethods map[string]MethodInstance
	SystemMethods map[string]MethodInstance
}

func NewContractInstance(contractInfo *context.ContractInfo) (*ContractInstance, error) {
	res := &ContractInstance{
		PublicMethods: make(map[string]MethodInstance),
		SystemMethods: make(map[string]MethodInstance),
	}
	for _, method := range contractInfo.PublicMethods {
		v := reflect.ValueOf(method)
		if v.Kind() != reflect.Func {
			return nil, errors.New("public method is not a valid func")
		}
		name := extractMethodName(runtime.FuncForPC(v.Pointer()).Name())
		res.PublicMethods[name] = method
	}
	for _, method := range contractInfo.SystemMethods {
		v := reflect.ValueOf(method)
		if v.Kind() != reflect.Func {
			return nil, errors.New("system method is not a valid func")
		}
		name := extractMethodName(runtime.FuncForPC(v.Pointer()).Name())
		res.SystemMethods[name] = method
	}
	return res, nil
}

func extractMethodName(fullPackageName string) string {
	parts := strings.Split(fullPackageName, ".")
	if len(parts) == 0 {
		return ""
	} else {
		return parts[len(parts)-1]
	}
}
