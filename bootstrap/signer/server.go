// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap"
	"github.com/orbs-network/orbs-network-go/services/signer"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
	"net/http"
)

type SignerServer struct {
	bootstrap.OrbsProcess
	service services.Vault
}

type SignerServerConfig interface {
	NodePrivateKey() primitives.EcdsaSecp256K1PrivateKey
	HttpAddress() string
}

func StartSignerServer(cfg SignerServerConfig, logger log.Logger) *SignerServer {
	_, cancel := context.WithCancel(context.Background())

	service := signer.NewService(cfg, logger)
	api := &api{
		service, logger,
	}

	httpServer, err := NewHttpServer(cfg.HttpAddress(), logger, func(router *http.ServeMux) {
		router.HandleFunc("/sign", api.SignHandler)
	})

	// Must find a better way
	if err != nil {
		panic(err)
	}

	s := &SignerServer{
		OrbsProcess: bootstrap.NewOrbsProcess(logger, cancel, httpServer),
		service:     service,
	}

	return s
}
