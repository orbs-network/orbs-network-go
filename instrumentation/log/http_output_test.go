package log

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"testing"
	"time"
)

type httpOutputHarness struct {
	port     uint16
	listener net.Listener
	router   *http.ServeMux
}

func newHttpHarness(handler http.Handler) *httpOutputHarness {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	port := uint16(20000 + r.Intn(40000))

	router := http.NewServeMux()
	router.Handle("/submit-logs", handler)

	return &httpOutputHarness{
		port:   port,
		router: router,
	}
}

func (h *httpOutputHarness) start(t *testing.T) {
	go func() {
		listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", h.port))
		h.listener = listener

		require.NoError(t, err, "failed to use http port")

		err = http.Serve(listener, h.router)
		require.NoError(t, err, "failed to server http requests")
	}()
}

func (h *httpOutputHarness) stop() {
	h.listener.Close()
}

func TestHttpWriter_Write(t *testing.T) {
	h := newHttpHarness(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		require.EqualValues(t, []byte("hello"), body)

		w.WriteHeader(200)
	}))
	h.start(t)
	defer h.stop()

	w := NewHttpWriter(fmt.Sprintf("http://localhost:%d/submit-logs", h.port))
	size, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	require.EqualValues(t, 5, size)
}
