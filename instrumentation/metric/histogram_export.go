package metric

import (
	"github.com/orbs-network/scribe/log"
	"strconv"
)

type histogramExport struct {
	Name    string
	Min     float64
	P50     float64
	P95     float64
	P99     float64
	Max     float64
	Avg     float64
	Samples int64
}

func (h histogramExport) LogRow() []*log.Field {
	if h.Samples == 0 {
		return nil
	}

	return []*log.Field{
		log.String("metric", h.Name),
		log.String("metric-type", "histogram"),
		log.Float64("min", h.Min),
		log.Float64("p50", h.P50),
		log.Float64("p95", h.P95),
		log.Float64("p99", h.P99),
		log.Float64("max", h.Max),
		log.Float64("avg", h.Avg),
		log.Int64("samples", h.Samples),
	}
}

func (h histogramExport) PrometheusRow() []*prometheusRow {
	name := h.PrometheusName()
	return []*prometheusRow{
		{name, "min", strconv.FormatFloat(h.Min, 'f', -1, 64)},
		{name, "median", strconv.FormatFloat(h.P50, 'f', -1, 64)},
		{name, "95p", strconv.FormatFloat(h.P95, 'f', -1, 64)},
		{name, "99p", strconv.FormatFloat(h.P99, 'f', -1, 64)},
		{name, "max", strconv.FormatFloat(h.Max, 'f', -1, 64)},
		{name, "avg", strconv.FormatFloat(h.Avg, 'f', -1, 64)},
		{name, "count", strconv.FormatFloat(float64(h.Samples), 'f', -1, 64)},
	}
}

func (h histogramExport) PrometheusType() string {
	return "histogram"
}

func (h histogramExport) PrometheusName() string {
	return prometheusName(h.Name)
}

func toMillis(nanoseconds int64) float64 {
	return floatToMillis(float64(nanoseconds))
}

func floatToMillis(nanoseconds float64) float64 {
	return nanoseconds / 1e+6
}
