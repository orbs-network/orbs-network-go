package httpserver

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type HttpServer interface {
	GracefulShutdown(timeout time.Duration)
}

type server struct {
	httpServer *http.Server
	reporting  instrumentation.Reporting
}

func NewHttpServer(address string, reporting instrumentation.Reporting, publicApi services.PublicApi) HttpServer {
	server := &server{
		httpServer: &http.Server{
			Addr:    address,
			Handler: createRouter(publicApi, reporting),
		},
		reporting: reporting,
	}
	go func() {
		server.httpServer.ListenAndServe() //TODO error on failed startup
	}()
	reporting.Info(fmt.Sprintf("server started on address %s", address))
	return server
}

//TODO extract commonalities between handlers
func createRouter(publicApi services.PublicApi, reporting instrumentation.Reporting) http.Handler {
	sendTransactionHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			clientRequest := client.SendTransactionRequestReader(bytes)
			if !clientRequest.IsValid() {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Transaction input is invalid"))
			} else {
				//TODO handle errors
				publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
				w.Header().Set("Content-Type", "application/octet-stream")
				//TODO return actual result once sendTranscation returns result.ClientOutput
				//w.Write()
			}
		}
	})

	callMethodHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			clientRequest := client.CallMethodRequestReader(bytes)
			if !clientRequest.IsValid() {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Call Method input is invalid"))
			} else {
				//TODO handle errors
				result, _ := publicApi.CallMethod(&services.CallMethodInput{ClientRequest: clientRequest})
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(result.ClientResponse.Raw())
			}
		}
	})

	router := http.NewServeMux()
	router.Handle("/api/send-transaction", report(reporting, sendTransactionHandler))
	router.Handle("/api/call-method", report(reporting, callMethodHandler))
	return router
}

func (s *server) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown(context.TODO()) //TODO timeout context
}

func report(reporting instrumentation.Reporting, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reporting.Infof("before %s", r.URL)
		defer func() {
			reporting.Infof("after %s, took %s", r.URL, time.Since(start))
		}()
		h.ServeHTTP(w, r)
	})
}
