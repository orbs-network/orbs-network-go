package gamma

import (
	"net/http"
	"time"
)

func (s *GammaServer) addGammaHandlers(router *http.ServeMux) {
	router.HandleFunc("/debug/gamma/shutdown", s.shutdownHandler)
	router.HandleFunc("/debug/gamma/inc-time", s.incrementTime)
}

func (s *GammaServer) shutdownHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	writer.Write([]byte(`{"status":"shutting down"}`))

	s.GracefulShutdown(1 * time.Second)
}

func (s *GammaServer) incrementTime(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := request.ParseForm(); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	seconds := request.Form.Get("seconds-to-add")
	if seconds == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

}
