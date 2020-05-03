// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package committee_systemcontract

import (
	"github.com/orbs-network/crypto-lib-go/crypto/ethereum/digest"
)

func makeNodeAddress(a int) []byte {
	addr := make([]byte, digest.NODE_ADDRESS_SIZE_BYTES)
	addr[0] = byte(a)
	return addr
}

func makeNodeAddressArray(n int) [][]byte {
	addrs := make([][]byte, 0, n)
	for i := 1; i <= n; i++ {
		addrs = append(addrs, makeNodeAddress(i))
	}
	return addrs
}
