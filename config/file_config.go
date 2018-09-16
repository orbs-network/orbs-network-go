package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NewFileConfig(source string) (NodeConfig, error) {
	cfg := EmptyConfig()

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(source), &data); err != nil {
		return nil, err
	}

	populateConfig(cfg, data)
	return cfg, nil
}

func convertKeyName(key string) string {
	return strings.ToUpper(strings.Replace(key, "-", "_", -1))
}

func populateConfig(cfg NodeConfig, data map[string]interface{}) {
	for key, value := range data {
		var duration time.Duration
		var numericValue uint32

		switch value.(type) {
		case float64:
			f64 := value.(float64)

			s := fmt.Sprintf("%.0f", f64)
			if i, err := strconv.Atoi(s); err == nil {
				numericValue = uint32(i)
			} else {
				// TODO handle error
			}
		case string:
			s := value.(string)

			if parsedDuration, err := time.ParseDuration(s); err == nil {
				duration = parsedDuration
			}
		}

		if numericValue != 0 {
			cfg.SetUint32(convertKeyName(key), numericValue)
		}

		if duration != 0 {
			cfg.SetDuration(convertKeyName(key), duration)
		}
	}
}
