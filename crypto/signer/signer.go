// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"net/http"
)

type Signer interface {
	Sign(ctx context.Context, input []byte) ([]byte, error)
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

func (c *local) Sign(ctx context.Context, input []byte) ([]byte, error) {
	return digest.SignAsNode(c.privateKey, input)
}

func NewSignerClient(address string) Signer {
	return &client{
		address: address,
	}
}

func (c *client) Sign(ctx context.Context, input []byte) ([]byte, error) {
	nodeSignInput := (&services.NodeSignInputBuilder{
		Data: input,
	}).Build()

	request, err := http.NewRequest("POST", c.address+"/sign", bytes.NewReader(nodeSignInput.Raw()))
	if err != nil {
		return nil, errors.Wrap(err, "error creating request to signer server")
	}
	request.Header.Set("Content-Type", "binary/octet-stream")
	if traceContext, _ := trace.FromContext(ctx); traceContext != nil {
		traceContext.WriteTraceToRequest(request)
	}

	client := http.DefaultClient
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "error sending request to signer server")
	}

	defer func() {
		err2 := response.Body.Close()
		if err2 != nil {
			log.Printf("Could not close response body.")
		}
	}()

	if response.StatusCode != http.StatusOK {
		return nil, errors.New("bad response code from signer server")
	}

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse signer server response")
	}

	return services.NodeSignOutputReader(data).Signature(), nil
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
