// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
	EventsMethods map[string]MethodInstance
}

func NewContractInstance(contractInfo *context.ContractInfo) (*ContractInstance, error) {
	res := &ContractInstance{
		PublicMethods: make(map[string]MethodInstance),
		SystemMethods: make(map[string]MethodInstance),
		EventsMethods: make(map[string]MethodInstance),
	}
	for _, method := range contractInfo.PublicMethods {
		name, err := GetContractMethodNameFromFunction(method)
		if err != nil {
			return nil, errors.Wrap(err, "invalid public method")
		}
		res.PublicMethods[name] = method
	}
	for _, method := range contractInfo.SystemMethods {
		name, err := GetContractMethodNameFromFunction(method)
		if err != nil {
			return nil, errors.Wrap(err, "invalid system method")
		}
		res.SystemMethods[name] = method
	}
	for _, method := range contractInfo.EventsMethods {
		name, err := GetContractMethodNameFromFunction(method)
		if err != nil {
			return nil, errors.Wrap(err, "invalid event method")
		}
		res.SystemMethods[name] = method
	}
	return res, nil
}

func GetContractMethodNameFromFunction(function interface{}) (string, error) {
	v := reflect.ValueOf(function)
	if v.Kind() != reflect.Func {
		return "", errors.New("did not receive a valid function")
	}
	fullPackageName := runtime.FuncForPC(v.Pointer()).Name()
	parts := strings.Split(fullPackageName, ".")
	if len(parts) == 0 {
		return "", errors.New("function name does not contain a valid package name")
	} else {
		return parts[len(parts)-1], nil
	}
}
