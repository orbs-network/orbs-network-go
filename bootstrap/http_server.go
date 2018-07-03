package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/events"
	"github.com/orbs-network/orbs-network-go/gossip"
	"github.com/orbs-network/orbs-network-go/loopcontrol"
	"github.com/orbs-network/orbs-network-go/publicapi"
	"github.com/orbs-network/orbs-network-go/types"
)

type HttpServer interface {
	Stop()
}

type httpServer struct {
	httpServer *http.Server
}

func NewHttpServer(address string, nodeId string, pauseableGossip gossip.Gossip, isLeader bool) HttpServer {
	storage := blockstorage.NewInMemoryBlockPersistence(nodeId)
	logger := events.NewStdoutLog()
	lc := loopcontrol.NewSimpleLoop()

	node := NewNode(pauseableGossip, storage, logger, lc, isLeader)

	server := &httpServer{
		httpServer: &http.Server{
			Addr:    address,
			Handler: createRouter(node.GetPublicApi()),
		},
	}

	go func() {
		server.httpServer.ListenAndServe() //TODO error on failed startup
	}()

	logger.Report(fmt.Sprintf("server started on address %s", address))

	return server

}
func createRouter(publicApi publicapi.PublicApi) http.Handler {
	sendTransactionHandler := func(w http.ResponseWriter, r *http.Request) {
		amountParam := r.URL.Query()["amount"][0]
		amount, _ := strconv.ParseInt(amountParam, 10, 32)
		publicApi.SendTransaction(&types.Transaction{
			Value: int(amount),
		})
	}

	callMethodHandler := func(w http.ResponseWriter, r *http.Request) {
		amount := publicApi.CallMethod()
		w.Write([]byte(fmt.Sprintf("%v", amount)))
	}

	router := http.NewServeMux()
	router.HandleFunc("/api/send_transaction", sendTransactionHandler)
	router.HandleFunc("/api/call_method", callMethodHandler)
	return router
}

func (s *httpServer) Stop() {
	s.httpServer.Shutdown(context.TODO()) //TODO context
}
