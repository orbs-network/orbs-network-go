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
	Error     string `json:",omitempty"`
	Payload   interface{}
}

func (s *HttpServer) getStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	statusString := s.getStatusWarningMessage()
	var errorString string
	if statusString != "OK" {
		errorString = statusString
	}

	status := StatusResponse{
		Timestamp: time.Now(),
		Status:    statusString,
		Error:     errorString,
		Payload:   s.metricRegistry.ExportAllNested(s.logger),
	}

	data, _ := json.MarshalIndent(status, "", "\t")

	_, err := w.Write(data)
	if err != nil {
		s.logger.Info("error writing index.json response", log.Error(err))

	}

}

func (s *HttpServer) getStatusWarningMessage() string {
	if metricObj := s.metricRegistry.Get("BlockStorage.FileSystemIndex.LastUpdateTime"); metricObj != nil {
		maxTimeSinceLastBlockStorageUpdate := s.config.TransactionPoolTimeBetweenEmptyBlocks().Nanoseconds() * 10
		if maxTimeSinceLastBlockStorageUpdate < 600000000 { // ten minutes
			maxTimeSinceLastBlockStorageUpdate = 600000000
		}
		lastBlockStorageUpdateTime := s.getGaugeValueFromMetrics("BlockStorage.FileSystemIndex.LastUpdateTime")
		if lastBlockStorageUpdateTime+maxTimeSinceLastBlockStorageUpdate < time.Now().UnixNano() {
			return "Last successful blockstorage update (including index update on boot) was too long ago"
		}
	}

	if metricObj := s.metricRegistry.Get("Management.Data.LastSuccessfulUpdateTime"); metricObj != nil {
		if len(s.config.ManagementFilePath()) != 0 && s.config.ManagementPollingInterval() > 0 {
			maxIntervalSinceLastSuccessfulManagementUpdate := int64(s.config.ManagementPollingInterval().Seconds()) * 20
			managementLastSuccessfullUpdate := s.getGaugeValueFromMetrics("Management.Data.LastSuccessfulUpdateTime")
			if managementLastSuccessfullUpdate == 0 {
				return "Management Service has never successfully updated"
			}
			if managementLastSuccessfullUpdate+maxIntervalSinceLastSuccessfulManagementUpdate < time.Now().Unix() {
				return "Last successful Management Service update was too long ago"
			}
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
