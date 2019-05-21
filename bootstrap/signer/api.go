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
