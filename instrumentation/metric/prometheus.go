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
