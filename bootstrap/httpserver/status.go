package httpserver

import (
	"encoding/json"
	"github.com/orbs-network/scribe/log"
	"net/http"
	"time"
)

type StatusResponse struct {
	Timestamp time.Time
	Status    string
	Error     string
	Payload   interface{}
}

func (s *HttpServer) getStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := StatusResponse{
		Timestamp: time.Now(),
		Status:    s.getStatusWarningMessage(),
		Payload:   s.metricRegistry.ExportAll(),
	}

	data, _ := json.MarshalIndent(status, "", "  ")

	_, err := w.Write(data)
	if err != nil {
		s.logger.Info("error writing index.json response", log.Error(err))

	}

}

func (s *HttpServer) getStatusWarningMessage() string {
	maxTimeSinceLastBlock := s.config.TransactionPoolTimeBetweenEmptyBlocks().Nanoseconds() * 10
	if maxTimeSinceLastBlock < 600000000 { // ten minutes
		maxTimeSinceLastBlock = 600000000
	}
	if s.getGaugeValueFromMetrics("ConsensusAlgo.LeanHelix.LastCommitted.TimeNano")+maxTimeSinceLastBlock <
		time.Now().UnixNano() {
		return "Last Successful Committed Block was too long ago"
	}

	if len(s.config.ManagementFilePath()) != 0 && s.config.ManagementPollingInterval() > 0 {
		maxIntervalSinceLastSuccessfulManagementUpdate := int64(s.config.ManagementPollingInterval().Seconds()) * 20
		if s.getGaugeValueFromMetrics("Management.Data.LastSuccessfulUpdateTime")+maxIntervalSinceLastSuccessfulManagementUpdate <
			time.Now().Unix() {
			return "Last Successful Management Update was too long ago"
		}
	}

	return "OK"
}

func (s *HttpServer) getGaugeValueFromMetrics(name string) (value int64) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("could not retrieve metric", log.String("metric", name))
		}
	}()

	rows := s.metricRegistry.Get(name).Export().LogRow()
	value = rows[len(rows)-1].Int
	return value
}
