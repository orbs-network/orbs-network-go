// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package signer

import (
	"github.com/orbs-network/orbs-network-go/services/signer"
	"github.com/orbs-network/scribe/log"
	"io/ioutil"
	"net/http"
)

type api struct {
	signer signer.Service
	logger log.Logger
}

func (a *api) SignHandler(writer http.ResponseWriter, request *http.Request) {
	input, err := ioutil.ReadAll(request.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		a.logger.Error("failed to sign payload")
		return
	}

	if signature, err := a.signer.Sign(input); err == nil {
		writer.Write(signature)
		a.logger.Info("successfully signed payload")
		return
	}

	writer.WriteHeader(http.StatusInternalServerError)
}
