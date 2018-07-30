package instrumentation

import (
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/crypto/base58"
	"strconv"
	"strings"
	"time"
)

type LogFormatter interface {
	FormatRow(level string, message string, params ...*Field) (formattedRow string)
}

type jsonFormatter struct {
}

func (j *jsonFormatter) FormatRow(level string, message string, params ...*Field) (formattedRow string) {
	logLine := make(map[string]interface{})

	logLine["level"] = level
	logLine["timestamp"] = float64(time.Now().UTC().UnixNano()) / NanosecondsInASecond
	logLine["message"] = message

	for _, param := range params {
		logLine[param.Key] = param.Value()
	}

	logLineAsJson, _ := json.Marshal(logLine)

	return string(logLineAsJson)
}

func NewJsonFormatter() LogFormatter {
	return &jsonFormatter{}
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
		value = param.String
	case NodeType:
		value = param.String
	case ServiceType:
		value = param.String
	case FunctionType:
		value = param.String
	case SourceType:
		value = param.String
	case IntType:
		value = strconv.FormatInt(param.Int, 10)
	case UintType:
		value = strconv.FormatUint(param.Uint, 10)
	case BytesType:
		value = string(base58.Encode(param.Bytes))
	case FloatType:
		value = strconv.FormatFloat(param.Float, 'f', 24, -1)
	case ErrorType:
		value = param.Error.Error()
	}

	builder.WriteString(param.Key)
	builder.WriteString(EQUALS)
	builder.WriteString(value)
	builder.WriteString(SPACE)
}

func cut(i int, params []*Field) []*Field {
	params[i] = params[len(params)-1] // Replace it with the last one.
	params = params[:len(params)-1]
	return params
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

func (j *humanReadableFormatter) FormatRow(level string, message string, params ...*Field) (formattedRow string) {
	logLine := make(map[string]interface{})

	logLine["level"] = level
	logLine["timestamp"] = float64(time.Now().UTC().UnixNano()) / NanosecondsInASecond
	logLine["message"] = message

	for _, param := range params {
		logLine[param.Key] = param.Value()
	}

	builder := strings.Builder{}

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)

	builder.WriteString(level)
	builder.WriteString(SPACE)
	builder.WriteString(timestamp)
	builder.WriteString(SPACE)

	builder.WriteString(message)
	builder.WriteString(SPACE)

	_, params = extractParamByType(params, NodeType, true, true, &builder)
	_, params = extractParamByType(params, ServiceType, true, true, &builder)
	functionParam, params := extractParamByType(params, FunctionType, false, true, nil)
	sourceParam, params := extractParamByType(params, SourceType, false, true, nil)

	for _, param := range params {
		printParam(&builder, param)
	}

	// append the function/source
	printParam(&builder, functionParam)
	printParam(&builder, sourceParam)

	return builder.String()
}

func NewHumanReadableFormatter() LogFormatter {
	return &humanReadableFormatter{}
}
