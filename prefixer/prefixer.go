package prefixer

import (
	"bufio"
	"bytes"
	"io"
)

type Prefixer struct {
	reader *bufio.Reader
	prefix string
}

func NewPrefix(reader io.Reader, prefix string) *Prefixer {
	if reader != nil {
		return &Prefixer{reader: bufio.NewReader(reader), prefix: prefix}
	} else {
		return &Prefixer{reader: bufio.NewReader(bytes.NewReader(nil)), prefix: prefix}
	}

}

func (p *Prefixer) Read(data []byte) (int, error) {
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
