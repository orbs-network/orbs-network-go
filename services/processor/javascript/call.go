// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

// +build jsprocessor

package javascript

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/ry/v8worker2"
)

func (s *service) processMethodCall(executionContextId primitives.ExecutionContextId, code string, methodName primitives.MethodName, args *protocol.ArgumentArray) (contractOutputArgs *protocol.ArgumentArray, contractOutputErr error, err error) {
	worker := v8worker2.New(func(msg []byte) []byte {
		s.logger.Info("bridge msg received", log.String("msg", hex.EncodeToString(msg)))
		contractOutputArgs, err = s.createMethodOutputArgs(msg)
		return nil
	})
	// defer worker.Dispose()

	contractOutputArgs = nil
	contractOutputErr = worker.Load("contract.js", s.wrapCodeForExecution(code, methodName, args))
	if contractOutputErr != nil {
		return nil, contractOutputErr, nil
	}

	return contractOutputArgs, contractOutputErr, err
}

const EXECUTION_WRAP_TEMPLATE = `
// sdk
%s

const res = (function($sdk, V8Worker2) {

// contract code
%s

// the call
return CounterFrom100.start();

})($sdk);

const buffer = new ArrayBuffer(4);
const view = new DataView(buffer);
view.setUint32(0, res, true);
V8Worker2.send(buffer);
`

func (s *service) wrapCodeForExecution(code string, methodName primitives.MethodName, args *protocol.ArgumentArray) string {
	return fmt.Sprintf(EXECUTION_WRAP_TEMPLATE, SDK_JS_IMPLEMENTATION, code)
}

func (s *service) createMethodOutputArgs(msg []byte) (*protocol.ArgumentArray, error) {
	if len(msg) != 4 {
		return nil, errors.Errorf("msg len is %d instead of 4", len(msg))
	}
	res := []*protocol.ArgumentBuilder{
		{Type: protocol.ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: uint64(binary.LittleEndian.Uint32(msg))},
	}
	return (&protocol.ArgumentArrayBuilder{
		Arguments: res,
	}).Build(), nil
}
