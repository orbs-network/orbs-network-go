package keys

import "testing"

func TestPrintAllAddresses(t *testing.T) {
	for i := 0; i < len(ecdsaSecp256K1KeyPairs); i++ {
		kp := EcdsaSecp256K1KeyPairForTests(i)
		t.Logf("Node %d Address: %s", i, kp.NodeAddress())
		t.Logf("Node %d Private: %s", i, kp.PublicKey())
		t.Log()
	}
}
