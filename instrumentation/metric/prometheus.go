package metric

import (
	"strconv"
	"strings"
)

type PrometheusRow struct {
	Name     string
	Quantile float64
	Value    string
}

func (r *PrometheusRow) name() string {
	return strings.ReplaceAll(r.Name, ".", "_")
}

func (r *PrometheusRow) quantiles() []string {
	if r.Quantile > 0 {
		return []string{`quantile="` + strconv.FormatFloat(r.Quantile, 'f', -1, 64) + `"`}
	}

	return nil
}

func (r *PrometheusRow) wrapParams() string {
	var params []string

	for _, q := range r.quantiles() {
		params = append(params, q)
	}

	if len(params) > 0 {
		return `{` + strings.Join(params, ",") + `}`
	}

	return ""
}

func (r *PrometheusRow) String() string {
	return r.name() + r.wrapParams() + " " + r.Value
}
