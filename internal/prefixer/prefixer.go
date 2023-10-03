package prefixer

import (
	"bufio"
	"bytes"
	"io"
)

type PrefixReader struct {
	reader *bufio.Reader
	prefix string
}

func NewPrefixReader(reader io.Reader, prefix string) *PrefixReader {
	if reader != nil {
		return &PrefixReader{reader: bufio.NewReader(reader), prefix: prefix}
	} else {
		return &PrefixReader{reader: bufio.NewReader(bytes.NewReader(nil)), prefix: prefix}
	}

}

func (p *PrefixReader) Read(data []byte) (int, error) {
	line, err := p.reader.ReadBytes('\n')
	if err != nil {
		//line = append([]byte(p.prefix), line...)
		n := copy(data, line)
		return n, err
	}
	line = append([]byte(p.prefix), line...)
	n := copy(data, line)
	return n, nil
}
