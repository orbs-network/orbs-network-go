package kms

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

type Signer interface {
	Sign(input []byte) ([]byte, error)
}

type local struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey
}

type client struct {
	address string
}

func NewLocalSigner(privateKey primitives.EcdsaSecp256K1PrivateKey) Signer {
	return &local{
		privateKey: privateKey,
	}
}

func (c *local) Sign(input []byte) ([]byte, error) {
	return digest.SignAsNode(c.privateKey, input)
}

func NewSignerClient(address string) Signer {
	return &client{
		address: address,
	}
}

func (c *client) Sign(input []byte) ([]byte, error) {
	response, err := http.Post(c.address+"/sign", "binary/octet-stream", bytes.NewReader(input))
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("bad response")
	}

	return ioutil.ReadAll(response.Body)
}

type SignerConfig interface {
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	SignerEndpoint() string
}

func GetSigner(config SignerConfig) Signer {
	if config.NodePrivateKey() != nil {
		return NewLocalSigner(config.NodePrivateKey())
	}

	if config.SignerEndpoint() != "" {
		return NewSignerClient(config.SignerEndpoint())
	}

	panic("bad private key configuration")
}
