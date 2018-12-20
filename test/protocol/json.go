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
