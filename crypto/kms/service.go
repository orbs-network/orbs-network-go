package kms

import (
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Service interface {
	Start() error
	Shutdown() error
	WaitUntilShutdown()
}

type service struct {
	privateKey primitives.EcdsaSecp256K1PrivateKey

	address string
	server  *http.Server

	logger log.Logger

	shutdownCondition *sync.Cond
}

func NewService(address string, privateKey primitives.EcdsaSecp256K1PrivateKey, logger log.Logger) Service {
	return &service{
		address:           address,
		privateKey:        privateKey,
		logger:            logger.WithTags(log.Service("signer")),
		shutdownCondition: sync.NewCond(&sync.Mutex{}),
	}
}

func (s *service) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	s.logger.Info("started signer", log.String("address", s.address))

	router := http.NewServeMux()
	router.HandleFunc("/sign", s.SignHandler)

	s.server = &http.Server{
		Handler: router,
	}

	// We prefer not to use `HttpServer.ListenAndServe` because we want to block until the socket is listening or exit immediately
	go s.server.Serve(httpserver.TcpKeepAliveListener{listener.(*net.TCPListener)})

	return nil
}

func (s *service) Shutdown() error {
	err := s.server.Close()
	s.shutdownCondition.Broadcast() // is there a better way?
	return err
}

func (s *service) SignHandler(writer http.ResponseWriter, request *http.Request) {
	input, err := ioutil.ReadAll(request.Body)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		s.logger.Error("failed to sign payload")
		return
	}

	if signature, err := digest.SignAsNode(s.privateKey, input); err == nil {
		writer.Write(signature)
		s.logger.Info("successfully signed payload")
		return
	}

	writer.WriteHeader(http.StatusInternalServerError)
}

func (s *service) WaitUntilShutdown() {
	// if waiting for shutdown, listen for sigint and sigterm
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	supervised.GoOnce(s.logger, func() {
		<-signalChan
		s.logger.Info("terminating node gracefully due to os signal received")
		s.Shutdown()
	})

	s.shutdownCondition.L.Lock()
	s.shutdownCondition.Wait()
	s.shutdownCondition.L.Unlock()
}
