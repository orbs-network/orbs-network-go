// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package httpserver

import (
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"net/http"
)

func (s *server) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte("User-agent: *\nDisallow: /\n"))
	if err != nil {
		s.logger.Info("error writing robots.txt response", log.Error(err))
	}
}

func (s *server) filterOn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, f := range s.logger.Filters() {
		if c, ok := f.(log.ConditionalFilter); ok {
			c.On()
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("filter on"))
}

func (s *server) filterOff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, f := range s.logger.Filters() {
		if c, ok := f.(log.ConditionalFilter); ok {
			c.Off()
		}
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("filter off"))
}

func (s *server) dumpMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	bytes, _ := json.Marshal(s.metricRegistry.ExportAll())
	_, err := w.Write(bytes)
	if err != nil {
		s.logger.Info("error writing response", log.Error(err))
	}
}

func (s *server) sendTransactionHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.SendTransactionRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received send-transaction", log.Stringable("request", clientRequest))
	result, err := s.publicApi.SendTransaction(r.Context(), &services.SendTransactionInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) sendTransactionAsyncHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.SendTransactionRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received send-transaction-async", log.Stringable("request", clientRequest))
	result, err := s.publicApi.SendTransactionAsync(r.Context(), &services.SendTransactionInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) runQueryHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.RunQueryRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received run-query", log.Stringable("request", clientRequest))
	result, err := s.publicApi.RunQuery(r.Context(), &services.RunQueryInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) getTransactionStatusHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.GetTransactionStatusRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received get-transaction-status", log.Stringable("request", clientRequest))
	result, err := s.publicApi.GetTransactionStatus(r.Context(), &services.GetTransactionStatusInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) getTransactionReceiptProofHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.GetTransactionReceiptProofRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received get-transaction-receipt-proof", log.Stringable("request", clientRequest))
	result, err := s.publicApi.GetTransactionReceiptProof(r.Context(), &services.GetTransactionReceiptProofInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) getBlockHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.GetBlockRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received get-block", log.Stringable("request", clientRequest))
	result, err := s.publicApi.GetBlock(r.Context(), &services.GetBlockInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}
