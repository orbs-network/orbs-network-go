package metric

import (
	"strconv"
	"strings"
)

type prometheusRow struct {
	name     string
	quantile float64
	value    string
}

type prometheusKeyValuePair struct {
	name  string
	value string
}

func prometheusName(name string) string {
	return strings.Replace(name, ".", "_", -1)
}

func (r *prometheusRow) wrapParams(pairs ...prometheusKeyValuePair) string {
	var params []string
	pairsCopy := pairs[:]

	if r.quantile > 0 {
		pairsCopy = append(pairsCopy, prometheusKeyValuePair{"quantile", strconv.FormatFloat(r.quantile, 'f', -1, 64)})
	}

	for _, p := range pairsCopy {
		params = append(params, p.name+`="`+p.value+`"`)
	}

	if len(params) > 0 {
		return `{` + strings.Join(params, ",") + `}`
	}

	return ""
}

func (r *prometheusRow) String(pairs ...prometheusKeyValuePair) string {
	return r.name + r.wrapParams(pairs...) + " " + r.value
}
