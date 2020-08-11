// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"fmt"
	"strconv"
	"strings"
)

/**
Format reference: https://prometheus.io/docs/instrumenting/exposition_formats/
For info on Prometheus labels, see: https://prometheus.io/docs/practices/naming/#labels
*/
func (r *inMemoryRegistry) ExportPrometheus() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	labelsString := r.labelsString()

	var rows []string
	for _, metric := range r.mu.metrics {
		rows = append(rows, metric.exportPrometheus(labelsString))
	}

	return strings.Join(rows, "")
}

func (r *inMemoryRegistry) labelsString() string {
	var lables []string
	if r.vcid > 0 {
		lables = append(lables, fmt.Sprintf("vcid=\"%s\"", strconv.FormatUint(uint64(r.vcid), 10)))
	}
	if r.nodeAddress != nil {
		lables = append(lables, fmt.Sprintf("node=\"%s\"", r.nodeAddress.String()))
	}
	return strings.Join(lables, ",")
}

// Note: in real life we have labels
func (g *Gauge) exportPrometheus(labelString string) string {
	typeRow := prometheusType(g.pName, "gauge")
	if len(labelString) > 0 {
		return typeRow + fmt.Sprintf("%s{%s} %s\n", g.pName, labelString, strconv.FormatInt(g.IntValue(), 10))
	}
	return typeRow + fmt.Sprintf("%s %s\n", g.pName, strconv.FormatInt(g.IntValue(), 10))
}

// Note: rate is not exported
func (r *Rate) exportPrometheus(labelString string) string {
	return ""
}

// Note: text is not exported
func (t *Text) exportPrometheus(labelString string) string {
	return ""
}

func prometheusName(name string) string {
	return strings.Replace(name, ".", "_", -1)
}

func prometheusType(name string, typeString string) string {
	return fmt.Sprintf("# TYPE %s %s\n", prometheusName(name), typeString)
}
