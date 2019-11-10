// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// Deprecated - use the protocol.ArgumentArrayFromNatives directly and deal with err
func ArgumentsArray(args ...interface{}) *protocol.ArgumentArray {
	res, err := protocol.ArgumentArrayFromNatives(args)
	if err != nil {
		panic(err.Error())
	}
	return res
}

// this is only for tests ... don't use in production code from here.
func VarsToSlice(args ...interface{}) []interface{} {
	return args
}
