// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

/// Test builders for: [][]byte

func EmptyPayloads(num int) [][]byte {
	res := [][]byte{}
	for i := 0; i < num; i++ {
		res = append(res, []byte{})
	}
	return res
}
