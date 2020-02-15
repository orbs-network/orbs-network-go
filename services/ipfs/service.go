// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ipfs

import (
	"context"
	ipfsClient "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"time"
)

var LogTag = log.Service("virtual-machine")

type service struct {
	logger log.Logger
	config config.IPFSConfig
}

type IPFSReadInput struct {
	Hash string
}

type IPFSReadOutput struct {
	Content []byte
}

type IPFSService interface {
	govnr.ShutdownWaiter
	Read(ctx context.Context, input *IPFSReadInput) (*IPFSReadOutput, error)
}

func NewIPFS(
	config config.IPFSConfig,
	logger log.Logger,
) IPFSService {
	s := &service{
		logger: logger.WithTags(LogTag),
		config: config,
	}

	return s
}

func (s *service) Read(ctx context.Context, input *IPFSReadInput) (*IPFSReadOutput, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			DisableKeepAlives: true,
		},
	}

	api, err := ipfsClient.NewURLApiWithClient(s.config.IPFSEndpoint(), client)
	if err != nil {
		return nil, errors.Errorf( "could not initialize ipfs client: %s", err)
	}

	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r, err := api.Object().Data(timeout, path.New(input.Hash))
	if err != nil {
		return nil, errors.Errorf("could not retrieve data from IPFS: %s", err)
	}

	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	content := raw[5:len(raw)-3]

	return &IPFSReadOutput{
		Content: content,
	}, nil
}

func (s *service) WaitUntilShutdown(shutdownContext context.Context) {
	s.logger.Info("shutting down")
}
