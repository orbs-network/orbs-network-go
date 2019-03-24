// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package digest

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
)

// don't need to provide hashed data as this function will SHA256
func SignAsNode(privateKey primitives.EcdsaSecp256K1PrivateKey, data []byte) (primitives.EcdsaSecp256K1Sig, error) {
	hashedData := hash.CalcSha256(data)
	return signature.SignEcdsaSecp256K1(privateKey, hashedData)
}

func VerifyNodeSignature(nodeAddress primitives.NodeAddress, data []byte, sig primitives.EcdsaSecp256K1Sig) error {
	if len(nodeAddress) != NODE_ADDRESS_SIZE_BYTES {
		return errors.Errorf("incorrect node address length. Expected=%d Actual=%d", NODE_ADDRESS_SIZE_BYTES, len(nodeAddress))
	}
	hashedData := hash.CalcSha256(data)
	publicKey, err := signature.RecoverEcdsaSecp256K1(hashedData, sig)
	if err != nil {
		return errors.Wrap(err, "RecoverEcdsaSecp256K1() failed")
	}
	recoveredNodeAddress := CalcNodeAddressFromPublicKey(publicKey)
	if !nodeAddress.Equal(recoveredNodeAddress) {
		return errors.Errorf("mismatched recovered node address. nodeAddress=%v recovered=%v", nodeAddress, recoveredNodeAddress)
	}
	return nil
}
