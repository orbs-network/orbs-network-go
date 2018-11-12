package log

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"net/http"
)

type httpOutput struct {
	formatter LogFormatter
	writer    io.Writer
}

func (out *httpOutput) Append(level string, message string, fields ...*Field) {
	logLine := out.formatter.FormatRow(level, message, fields...)
	fmt.Fprintln(out.writer, logLine)
}

func NewHttpOutput(writer io.Writer, formatter LogFormatter) Output {
	return &basicOutput{formatter, writer}
}

type httpWriter struct {
	url string
}

func (w *httpWriter) Write(p []byte) (n int, err error) {
	reader := bytes.NewReader(p)
	size := reader.Len()
	resp, err := http.Post(w.url, "application/json", reader)
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, errors.Errorf("Failed to send logs: %d, %s", resp.StatusCode, err)
	}

	if err != nil {
		return 0, errors.Errorf("Failed to send logs: %d, %s", resp.StatusCode, err)
	}

	return size, err
}

func NewHttpWriter(url string) io.Writer {
	return &httpWriter{url}
}
