package gamma

import (
	"net/http"
	"time"
)

func (s *GammaServer) addGammaHandlers(router *http.ServeMux) {
	router.HandleFunc("/debug/gamma/shutdown", s.shutdownHandler)
}

func (s *GammaServer) shutdownHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	writer.Write([]byte(`{"status":"shutting down"}`))

	s.GracefulShutdown(1 * time.Second)
}
