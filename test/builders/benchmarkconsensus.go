package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type committed struct {
	signerPrivateKey  primitives.Ed25519PrivateKey
	tempInvalidSigner bool //TODO: kill me
	status            *gossipmessages.BenchmarkConsensusStatusBuilder
	sender            *gossipmessages.SenderSignatureBuilder
}

func BenchmarkConsensusCommittedMessage() *committed {
	return &committed{
		status: &gossipmessages.BenchmarkConsensusStatusBuilder{
			LastCommittedBlockHeight: 0,
		},
		sender: &gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: []byte{0x66},
			Signature:       nil,
		},
	}
}

func (c *committed) WithLastCommittedHeight(blockHeight primitives.BlockHeight) *committed {
	c.status.LastCommittedBlockHeight = blockHeight
	return c
}

func (c *committed) WithSenderSignature(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *committed {
	c.signerPrivateKey = privateKey
	c.sender.SenderPublicKey = publicKey
	return c
}

func (c *committed) WithInvalidSenderSignature(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *committed {
	c.tempInvalidSigner = true
	return c.WithSenderSignature(make([]byte, len(privateKey)), publicKey)
}

func (c *committed) Build() *gossipmessages.BenchmarkConsensusCommittedMessage {
	statusBuilt := c.status.Build()
	signedData := hash.CalcSha256(statusBuilt.Raw())
	sig := signature.SignEd25519(c.signerPrivateKey, signedData)
	if c.tempInvalidSigner {
		sig[0] ^= 0xaa // TODO: kill me
	}
	c.sender.Signature = sig
	return &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: statusBuilt,
		Sender: c.sender.Build(),
	}
}
