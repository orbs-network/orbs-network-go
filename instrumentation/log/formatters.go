// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package log

import (
	"encoding/hex"
	"encoding/json"
	"github.com/go-playground/ansi"
	"strconv"
	"strings"
	"time"
)

type LogFormatter interface {
	FormatRow(timestamp time.Time, level string, message string, params ...*Field) (formattedRow string)
}

type jsonFormatter struct {
	timestampColumn string
}

const DEFAULT_TIMESTAMP_COLUMN = "timestamp"
const TIMESTAMP_FORMAT = "2006-01-02T15:04:05.999999999Z"

func (j *jsonFormatter) FormatRow(timestamp time.Time, level string, message string, params ...*Field) (formattedRow string) {
	logLine := make(map[string]interface{})

	logLine["level"] = level
	logLine[j.timestampColumn] = timestamp.UTC().Format(TIMESTAMP_FORMAT)
	logLine["message"] = message

	logFields(params, logLine)

	logLineAsJson, _ := json.Marshal(logLine)

	return string(logLineAsJson)
}

func logFields(params []*Field, logLine map[string]interface{}) {
	for _, param := range params {
		logLine[param.Key] = param.Value()
	}
}

func NewJsonFormatter() *jsonFormatter {
	return &jsonFormatter{
		timestampColumn: DEFAULT_TIMESTAMP_COLUMN,
	}
}

func (j *jsonFormatter) WithTimestampColumn(column string) *jsonFormatter {
	j.timestampColumn = column
	return j
}

type humanReadableFormatter struct {
}

const (
	SPACE  = " "
	EQUALS = "="
)

func findFieldByType(fieldType FieldType, params []*Field) (index int, result *Field) {
	for idx, param := range params {
		if param.Type == fieldType {
			return idx, param
		}
	}

	return -1, nil
}

func printParam(builder *strings.Builder, param *Field) {
	if param == nil {
		return
	}

	var value string

	switch param.Type {
	case StringType:
		value = param.StringVal
	case NodeType:
		value = param.StringVal
	case ServiceType:
		value = param.StringVal
	case FunctionType:
		value = param.StringVal
	case SourceType:
		value = param.StringVal
	case IntType:
		value = strconv.FormatInt(param.Int, 10)
	case UintType:
		value = strconv.FormatUint(param.Uint, 10)
	case BytesType:
		value = hex.EncodeToString(param.Bytes)
	case FloatType:
		value = strconv.FormatFloat(param.Float, 'f', -1, 64)
	case ErrorType:
		if param.Error != nil {
			value = param.Error.Error()
		} else {
			value = "<nil>"
		}
	case StringArrayType:
		json, err := json.MarshalIndent(param.StringArray, "", "\t")
		if err != nil {
			value = ""
		} else {
			value = string(json)
		}
	}

	builder.WriteString(param.Key)
	builder.WriteString(EQUALS)
	builder.WriteString(value)
	builder.WriteString(SPACE)
}

func cut(i int, params []*Field) []*Field {
	copy(params[i:], params[i+1:])
	params[len(params)-1] = nil
	params = params[:len(params)-1]
	return params
}

func extractParamByTypePrintAndRemove(params []*Field, ft FieldType, builder *strings.Builder) (*Field, []*Field) {
	return extractParamByType(params, ft, true, true, builder)
}

func extractParamByTypeAndRemove(params []*Field, ft FieldType) (*Field, []*Field) {
	return extractParamByType(params, ft, false, true, nil)
}

func extractParamByType(params []*Field, ft FieldType, shouldPrint, shouldRemove bool, builder *strings.Builder) (*Field, []*Field) {
	if idx, param := findFieldByType(ft, params); param != nil {
		if shouldPrint {
			printParam(builder, param)
		}
		if shouldRemove {
			params = cut(idx, params)
		}

		return param, params
	}

	return nil, params
}

func extractParamByConditionAndRemove(params []*Field, condition func(param *Field) bool) (results []*Field, newParams []*Field) {
	for _, param := range params {
		if condition(param) {
			results = append(results, param)
		} else {
			newParams = append(newParams, param)
		}
	}

	return results, newParams
}

func (j *humanReadableFormatter) FormatRow(timestamp time.Time, level string, message string, params ...*Field) (formattedRow string) {
	builder := strings.Builder{}
	var mutableParams = make([]*Field, len(params)) // this is needed because extractParamByTypePrintAndRemove mutates the array
	copy(mutableParams, params)

	ts := timestamp.UTC().Format("2006-01-02T15:04:05.000000Z07:00")

	builder.WriteString(colorize(mutableParams))
	builder.WriteString(level)
	builder.WriteString(SPACE)
	builder.WriteString(ts)
	builder.WriteString(SPACE)

	builder.WriteString(message)
	builder.WriteString(SPACE)

	_, mutableParams = extractParamByTypePrintAndRemove(mutableParams, NodeType, &builder)
	_, mutableParams = extractParamByTypePrintAndRemove(mutableParams, ServiceType, &builder)
	functionParam, mutableParams := extractParamByTypeAndRemove(mutableParams, FunctionType)
	sourceParam, mutableParams := extractParamByTypeAndRemove(mutableParams, SourceType)
	underscoreParams, mutableParams := extractParamByConditionAndRemove(mutableParams, func(param *Field) bool {
		return strings.Index(param.Key, "_") == 0
	})

	for _, p := range mutableParams {
		printParam(&builder, p)
	}

	// append the function/source
	printParam(&builder, functionParam)
	printParam(&builder, sourceParam)

	for _, param := range underscoreParams {
		printParam(&builder, param)
	}
	return builder.String()
}

func colorize(fields []*Field) string {
	colors := []string{ansi.Cyan, ansi.Yellow, ansi.LightBlue, ansi.Magenta, ansi.LightYellow, ansi.LightRed, ansi.LightGreen, ansi.LightMagenta, ansi.Green}
	for _, f := range fields {
		if f.Key == "request-id" {
			fourthBeforeLastChar := int(f.StringVal[len(f.StringVal)-4])
			return colors[fourthBeforeLastChar%len(colors)]
		}
	}

	return ""
}

func NewHumanReadableFormatter() LogFormatter {
	return &humanReadableFormatter{}
}
