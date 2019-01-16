package log

import (
	"io"
	"os"
)

type rotatingFileWriter struct {
	f *os.File
}

type RotatingFileWriter interface {
	io.Writer
	Rotate() error
}

func NewRotatingFileWriter(f *os.File) RotatingFileWriter {
	return &rotatingFileWriter{
		f: f,
	}
}

func (w *rotatingFileWriter) Write(p []byte) (n int, err error) {
	return w.f.Write(p)
}

func (w *rotatingFileWriter) Rotate() error {
	if err := w.f.Truncate(0); err != nil {
		return err
	} else {
		w.f.Seek(0, 0)
	}

	return nil
}
