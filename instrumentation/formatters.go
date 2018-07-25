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

func findFieldByType(fieldType int, params []*Field) (index int, result *Field) {
	for idx, param := range params {
		if param.Type == FieldType(fieldType) {
			return idx, param
		}
	}

	return -1, nil
}

func printParam(builder *strings.Builder, param *Field) {
	var value string

	switch param.Type {
	case StringType:
		value = param.String
	case NodeType:
		value = param.String
	case ServiceType:
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
	case BlockHeightType:
		value = strconv.FormatUint(param.Uint, 10)
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

	if idx, param := findFieldByType(NodeType, params); param != nil {
		printParam(&builder, param)
		params = cut(idx, params)
	}

	if idx, param := findFieldByType(ServiceType, params); param != nil {
		printParam(&builder, param)
		params = cut(idx, params)
	}

	if idx, param := findFieldByType(BlockHeightType, params); param != nil {
		printParam(&builder, param)
		params = cut(idx, params)
	}

	for _, param := range params {
		printParam(&builder, param)
	}

	return builder.String()
}

func NewHumanReadableFormatter() LogFormatter {
	return &humanReadableFormatter{}
}
