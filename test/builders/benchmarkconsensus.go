package builders

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
)

type committed struct {
	messageSigner primitives.Ed25519PrivateKey
	status        *gossipmessages.BenchmarkConsensusStatusBuilder
	sender        *gossipmessages.SenderSignatureBuilder
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
	c.messageSigner = privateKey
	c.sender.SenderPublicKey = publicKey
	return c
}

func (c *committed) WithInvalidSenderSignature(privateKey primitives.Ed25519PrivateKey, publicKey primitives.Ed25519PublicKey) *committed {
	return c.WithSenderSignature(make([]byte, len(privateKey)), publicKey)
}

func (c *committed) Build() *gossipmessages.BenchmarkConsensusCommittedMessage {
	statusBuilt := c.status.Build()
	signedData := hash.CalcSha256(statusBuilt.Raw())
	sig, err := signature.SignEd25519(c.messageSigner, signedData)
	if err != nil {
		panic(err)
	}
	c.sender.Signature = sig
	return &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: statusBuilt,
		Sender: c.sender.Build(),
	}
}
