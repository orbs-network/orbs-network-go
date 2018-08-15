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
	logLine["timestamp"] = time.Now().UTC().UnixNano()
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
	case StringArrayType:
		x := make(map[int]string)
		for i, v := range param.StringArray {
			x[i] = v
		}
		json, err := json.MarshalIndent(x, "", " ")
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

func (j *humanReadableFormatter) FormatRow(level string, message string, params ...*Field) (formattedRow string) {
	builder := strings.Builder{}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.999999Z07:00")

	builder.WriteString(level)
	builder.WriteString(SPACE)
	builder.WriteString(timestamp)
	builder.WriteString(SPACE)

	builder.WriteString(message)
	builder.WriteString(SPACE)

	_, params = extractParamByTypePrintAndRemove(params, NodeType, &builder)
	_, params = extractParamByTypePrintAndRemove(params, ServiceType, &builder)
	functionParam, params := extractParamByTypeAndRemove(params, FunctionType)
	sourceParam, params := extractParamByTypeAndRemove(params, SourceType)

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
