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
	leanHelixLastCommitteed := s.getGaugeValueFromMetrics("ConsensusAlgo.LeanHelix.LastCommitted.TimeNano")
	if leanHelixLastCommitteed == 0 {
		return "LeanHelix Service has not committed any blocks yet"
	}
	if leanHelixLastCommitteed + maxTimeSinceLastBlock < time.Now().UnixNano() {
		return "Last Successful Committed Block was too long ago"
	}

	if len(s.config.ManagementFilePath()) != 0 && s.config.ManagementPollingInterval() > 0 {
		maxIntervalSinceLastSuccessfulManagementUpdate := int64(s.config.ManagementPollingInterval().Seconds()) * 20
		managementLastSuccessfullUpdate := s.getGaugeValueFromMetrics("Management.Data.LastSuccessfulUpdateTime")
		if managementLastSuccessfullUpdate == 0 {
			return "Management Service has never successfully updated"
		}
		if managementLastSuccessfullUpdate + maxIntervalSinceLastSuccessfulManagementUpdate < time.Now().Unix() {
			return "Last successful Management Service update was too long ago"
		}
	}

	return "OK"
}

func (s *HttpServer) getGaugeValueFromMetrics(name string) int64 {
	metricObj := s.metricRegistry.Get(name)
	if metricObj == nil {
		return 0
	}
	if value, ok := metricObj.Value().(int64); !ok {
		return 0
	} else {
		return value
	}
}
