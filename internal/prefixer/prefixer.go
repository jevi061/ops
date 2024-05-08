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
	b, err := p.reader.ReadByte()
	sb := []byte{b}
	if err != nil {
		n := copy(data, sb)
		return n, err
	}
	if b == '\n' {
		sb = append(sb, []byte(p.prefix)...)
	}
	n := copy(data, sb)
	return n, nil
}
