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
*/

// Note: in real life we have labels
func (g *Gauge) exportPrometheus(labelString string) string {
	typeRow := prometheusType(prometheusName(g.name), "gauge")
	if len(labelString) > 0 {
		return typeRow + fmt.Sprintf("%s{%s} %s\n", prometheusName(g.name), labelString, strconv.FormatInt(g.IntValue(), 10))
	}
	return typeRow + fmt.Sprintf("%s %s\n", prometheusName(g.name), strconv.FormatInt(g.IntValue(), 10))
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
