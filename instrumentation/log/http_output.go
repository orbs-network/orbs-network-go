package log

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"sync"
	"time"
)

type row struct {
	level     string
	timestamp time.Time
	message   string
	fields    []*Field
}

type bulkOutput struct {
	formatter LogFormatter
	writer    io.Writer

	bulkSize int
	delay    time.Duration

	lock *sync.Mutex
	logs []*row
}

func (out *bulkOutput) Append(level string, message string, fields ...*Field) {
	row := &row{level, time.Now(), message, fields}

	out.lock.Lock()
	out.logs = append(out.logs, row)
	out.lock.Unlock()

	out.flushIfNeeded()
}

func (out *bulkOutput) flushIfNeeded() {
	out.lock.Lock()
	defer out.lock.Unlock()

	if len(out.logs) >= out.bulkSize {
		b := new(bytes.Buffer)

		for _, row := range out.logs {
			b.Write([]byte(out.formatter.FormatRow(row.timestamp, row.level, row.message, row.fields...)))
			b.Write([]byte("\n"))
		}

		out.logs = nil

		go func() {
			if n, err := out.writer.Write(b.Bytes()); err != nil {
				fmt.Println(fmt.Sprintf("%s failed to send logs via http, %d bytes lost: %s", time.Now().String(), n, err))
			}
		}()
	}
}

func NewBulkOutput(writer io.Writer, formatter LogFormatter, bulkSize int) Output {
	return &bulkOutput{
		formatter: formatter,
		writer:    writer,
		bulkSize:  bulkSize,
		lock:      &sync.Mutex{},
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
