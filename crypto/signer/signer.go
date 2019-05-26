// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"bytes"
	"github.com/orbs-network/orbs-network-go/config"
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

func New(cfg config.SignerConfig) (Signer, error) {
	if cfg.NodePrivateKey() != nil {
		return NewLocalSigner(cfg.NodePrivateKey()), nil
	}

	if cfg.SignerEndpoint() != "" {
		return NewSignerClient(cfg.SignerEndpoint()), nil
	}

	return nil, errors.New("bad private key configuration: both private key and signer endpoint were not set")
}
