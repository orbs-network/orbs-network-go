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

type PrometheusKeyValuePair struct {
	Name  string
	Value string
}

func (r *PrometheusRow) name() string {
	return strings.Replace(r.Name, ".", "_", -1)
}

func (r *PrometheusRow) quantiles() []string {

	return nil
}

func (r *PrometheusRow) wrapParams(pairs ...PrometheusKeyValuePair) string {
	var params []string

	for _, p := range pairs {
		params = append(params, p.Name+`="`+p.Value+`"`)
	}

	if r.Quantile > 0 {
		params = append(params, `quantile="`+strconv.FormatFloat(r.Quantile, 'f', -1, 64)+`"`)
	}

	if len(params) > 0 {
		return `{` + strings.Join(params, ",") + `}`
	}

	return ""
}

func (r *PrometheusRow) String(pairs ...PrometheusKeyValuePair) string {
	return r.name() + r.wrapParams(pairs...) + " " + r.Value
}
