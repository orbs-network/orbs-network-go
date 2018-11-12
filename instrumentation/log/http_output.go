package log

import (
	"bytes"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"strings"
	"time"
)

type row struct {
	level     string
	timestamp time.Time
	message   string
	fields    []*Field
}

type httpOutput struct {
	formatter LogFormatter
	writer    io.Writer

	bulkSize int
	delay    time.Duration

	logs    []*row
	updated time.Time
}

func (out *httpOutput) Append(level string, message string, fields ...*Field) {
	timestamp := time.Now()
	row := &row{level, timestamp, message, fields}

	if len(out.logs) >= out.bulkSize || (out.updated.UnixNano()-timestamp.UnixNano()) >= out.delay.Nanoseconds() {
		lines := []string{}
		for _, row := range out.logs {
			// FIXME timestamp problem
			lines = append(lines, out.formatter.FormatRow(row.level, row.message, row.fields...))
		}

		go out.writer.Write([]byte(strings.Join(lines, "\n")))
	}

	out.logs = append(out.logs, row)
	out.updated = timestamp
}

func NewHttpOutput(writer io.Writer, formatter LogFormatter, bulkSize int, maxDelay time.Duration) Output {
	return &httpOutput{
		formatter: formatter,
		writer:    writer,
		bulkSize:  bulkSize,
		delay:     maxDelay,
	}
}

type httpWriter struct {
	url string
}

func (w *httpWriter) Write(p []byte) (n int, err error) {
	reader := bytes.NewReader(p)
	size := reader.Len()
	resp, err := http.Post(w.url, "application/json", reader)

	if err != nil {
		return 0, errors.Errorf("Failed to send logs: %s", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, errors.Errorf("Failed to send logs: %d, %s", resp.StatusCode, err)
	}

	return size, err
}

func NewHttpWriter(url string) io.Writer {
	return &httpWriter{url}
}
