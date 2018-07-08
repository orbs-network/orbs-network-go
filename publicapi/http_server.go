package publicapi

import (
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"net/http"
	"fmt"
	"context"
	"time"
	"io/ioutil"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type HttpServer interface {
	GracefulShutdown(timeout time.Duration)
}

type httpServer struct {
	httpServer *http.Server
}

func NewHttpServer(address string, logger instrumentation.Reporting, publicApi services.PublicApi) HttpServer {

	server := &httpServer{
		httpServer: &http.Server {
			Addr:    address,
			Handler: createRouter(publicApi),
		},
	}

	go func() {
		server.httpServer.ListenAndServe() //TODO error on failed startup
	}()

	logger.Info(fmt.Sprintf("server started on address %s", address))

	return server

}

func createRouter(publicApi services.PublicApi) http.Handler {
	sendTransactionHandler := func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			input := services.SendTransactionInputReader(bytes)
			if !input.IsValid() {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Transaction input is invalid"))
			} else {
				//TODO handle errors
				result, _ := publicApi.SendTransaction(input)
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(result.Raw())
			}
		}
	}

	callMethodHandler := func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			input := services.CallMethodInputReader(bytes)
			if !input.IsValid() {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Call Method input is invalid"))
			} else {
				//TODO handle errors
				result, _ := publicApi.CallMethod(input)
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Write(result.Raw())
			}
		}
	}

	router := http.NewServeMux()
	router.HandleFunc("/api/send-transaction", sendTransactionHandler)
	router.HandleFunc("/api/call-method", callMethodHandler)
	return router
}

func (s *httpServer) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown(context.TODO()) //TODO timeout context
}

