// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package keys

import "testing"

func TestPrintAllAddresses(t *testing.T) {
	for i := 0; i < len(ecdsaSecp256K1KeyPairs); i++ {
		kp := EcdsaSecp256K1KeyPairForTests(i)
		t.Logf("Node %d Address: %s", i, kp.NodeAddress())
		t.Logf("Node %d Private: %s", i, kp.PrivateKey())
		t.Log()
	}
}
