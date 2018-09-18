package httpserver

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"strconv"
	"strings"
)

func BenchmarkServerCallMethod(b *testing.B) {
	logger := log.GetLogger().WithOutput()
	mockApi := getPapiMock()

	address, url := getDestiantions(8080)

	s := NewHttpServer(address, logger, mockApi)
	defer s.GracefulShutdown(time.Second)

	webClient := &http.Client{}

	request := (&client.CallMethodRequestBuilder{
		Transaction: &protocol.TransactionBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", url, bytes.NewReader(request.Raw()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendRequest(webClient, req)
	}
	b.StopTimer()
}

func BenchmarkFastServerCallMethod(b *testing.B) {
	logger := log.GetLogger().WithOutput()
	mockApi := getPapiMock()

	address, url := getDestiantions(8081)

	s := NewFastHttpServer(address, logger, mockApi)
	defer s.GracefulShutdown(time.Second)

	webClient := &http.Client{}

	request := (&client.CallMethodRequestBuilder{
		Transaction: &protocol.TransactionBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", url, bytes.NewReader(request.Raw()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendRequest(webClient, req)
	}
	b.StopTimer()
}

func getPapiMock() *services.MockPublicApi {
	papiMock := &services.MockPublicApi{}
	response := &client.CallMethodResponseBuilder{
		RequestStatus:       protocol.REQUEST_STATUS_COMPLETED,
		OutputArgumentArray: nil,
		CallMethodResult:    protocol.EXECUTION_RESULT_SUCCESS,
		BlockHeight:         1,
		BlockTimestamp:      primitives.TimestampNano(time.Now().Nanosecond()),
	}
	papiMock.When("CallMethod", mock.Any).Times(1).Return(&services.CallMethodOutput{ClientResponse: response.Build()})
	return papiMock
}

func sendRequest(client *http.Client, request *http.Request) {
	res, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	if res.StatusCode != 200 {
		panic("request failed")
	}

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	err = res.Body.Close()
	if err != nil {
		panic(err)
	}
}

func getDestiantions(port int) (address string, url string) {
	address = strings.Join([]string{"127.0.0.1", ":", strconv.Itoa(port)}, "")
	url = strings.Join([]string{"http://", address, "/api/v1/call-method"}, "")
	return address, url
}
