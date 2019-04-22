// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package digest

import (
	"fmt"
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

func VerifyNodeSignature(actualNodeAddress primitives.NodeAddress, dataToVerify []byte, sig primitives.EcdsaSecp256K1Sig) error {
	if len(actualNodeAddress) != NODE_ADDRESS_SIZE_BYTES {
		return errors.Errorf("incorrect actual node address length. ExpectedLen=%d ActualLen=%d", NODE_ADDRESS_SIZE_BYTES, len(actualNodeAddress))
	}
	hashedDataToVerify := hash.CalcSha256(dataToVerify)
	recoveredPublicKey, err := signature.RecoverEcdsaSecp256K1(hashedDataToVerify, sig)
	if err != nil {
		return errors.Wrap(err, "RecoverEcdsaSecp256K1() failed")
	}
	recoveredNodeAddress := CalcNodeAddressFromPublicKey(recoveredPublicKey)
	msg := fmt.Sprintf("actualNodeAddress=%v recoveredNodeAddress=%v sig=%s dataToVerify=%v hashedDataToVerify=%v recoveredPublicKey=%v",
		actualNodeAddress, recoveredNodeAddress, sig, dataToVerify, hashedDataToVerify, recoveredPublicKey)

	if !actualNodeAddress.Equal(recoveredNodeAddress) {
		return errors.Errorf("mismatched actual and calculated node address: %s", msg)
	}
	return nil
}
