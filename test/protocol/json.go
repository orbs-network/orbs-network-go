// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package protocol

import (
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/orbs-network-go/crypto/merkle"
)

func numberToJSON(num interface{}) string {
	return fmt.Sprintf("%d", num)
}

func bytesToJSON(buf []byte) string {
	return hex.EncodeToString(buf)
}

func merkleProofToJSON(proof merkle.OrderedTreeProof) []string {
	res := []string{}
	for _, hash := range proof {
		res = append(res, bytesToJSON(hash))
	}
	return res
}
