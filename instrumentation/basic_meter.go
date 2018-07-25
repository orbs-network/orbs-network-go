package instrumentation

import (
	"fmt"
	"strings"
	"time"
)

type basicMeter struct {
	name   string
	start  int64
	end    int64
	logger BasicLogger

	params []*Field
}

type BasicMeter interface {
	Done()
}

func (m *basicMeter) Done() {
	m.end = time.Now().UnixNano()
	diff := float64(m.end-m.start) / NanosecondsInASecond

	var names []string
	for _, prefix := range m.logger.Prefixes() {
		if prefix.Type == NodeType {
			continue
		}
		names = append(names, fmt.Sprintf("%s", prefix.Value()))
	}

	names = append(names, m.name)
	metricName := strings.Join(names, "-")

	metricParams := append(m.params, Float64("process-time", diff))
	m.logger.Metric(metricName, metricParams...)
}
