package multiwritecloser

import (
	"errors"
	"io"
)

type MultiWriteCloser struct {
	writeclosers []io.WriteCloser
}

func NewMultiWriteCloser(wrs ...io.WriteCloser) io.WriteCloser {
	return &MultiWriteCloser{writeclosers: wrs}
}

func (m *MultiWriteCloser) Write(p []byte) (n int, err error) {
	for _, w := range m.writeclosers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, errors.New("short write")
		}
	}
	return n, nil
}
func (m *MultiWriteCloser) Close() error {
	for _, w := range m.writeclosers {
		if err := w.Close(); err != nil {
			return err
		}
	}
	return nil
}
