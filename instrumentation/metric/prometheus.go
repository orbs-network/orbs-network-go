// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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

func (r *prometheusRow) wrapLabels(pairs ...prometheusKeyValuePair) string {
	var labels []string
	pairsCopy := pairs[:]

	if r.quantile >= 0 {
		pairsCopy = append(pairsCopy, prometheusKeyValuePair{"quantile", quantileAsStr(r.quantile)})
	}

	for _, p := range pairsCopy {
		labels = append(labels, p.name+`="`+p.value+`"`)
	}

	if len(labels) > 0 {
		return `{` + strings.Join(labels, ",") + `}`
	}

	return ""
}

func quantileAsStr(quantile float64) string {
	return strconv.FormatFloat(quantile, 'f', -1, 64)
}

func (r *prometheusRow) String(labelKeyValues ...prometheusKeyValuePair) string {
	return r.name + r.wrapLabels(labelKeyValues...) + " " + r.value
}
