package leanhelixconsensus

import (
	"github.com/orbs-network/lean-helix-go"
	lhprimitives "github.com/orbs-network/lean-helix-go/primitives"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

type keyManager struct {
	publicKey  primitives.Ed25519PublicKey
	privateKey primitives.Ed25519PrivateKey
}

func NewKeyManager(publicKey primitives.Ed25519PublicKey, privateKey primitives.Ed25519PrivateKey) *keyManager {
	return &keyManager{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

func (k *keyManager) Sign(content []byte) []byte {
	sig, _ := signature.SignEd25519(k.privateKey, content)
	return sig
}

func (k *keyManager) Verify(content []byte, sender *leanhelix.SenderSignature) bool {

	return signature.VerifyEd25519(primitives.Ed25519PublicKey(sender.SenderPublicKey()), content, primitives.Ed25519Sig(sender.Signature()))
}

func (k *keyManager) MyPublicKey() lhprimitives.Ed25519PublicKey {
	return lhprimitives.Ed25519PublicKey(k.publicKey)
}
