package git

import (
	"bufio"
	"compress/zlib"
	"io"
	"os"
	"path/filepath"
)

type looseObjectEntry struct {
	f   *os.File
	zr  io.ReadCloser
	br  *bufio.Reader
	typ string
}

func newLooseObjectEntry(root string, id SHA1) (*looseObjectEntry, error) {
	s := id.String()
	path := filepath.Join(root, "objects", s[:2], s[2:])

	e := new(looseObjectEntry)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	e.f = file

	zr, err := zlib.NewReader(file)
	if err != nil {
		file.Close()
		return nil, err
	}
	e.zr = zr

	var bs []byte
	e.br = bufio.NewReader(zr)
	if bs, err = e.br.ReadBytes(' '); err != nil {
		e.Close()
		return nil, err
	}
	e.typ = string(bs[:len(bs)-1])

	if _, err = e.br.ReadBytes(0); err != nil {
		e.Close()
		return nil, err
	}
	return e, nil
}

func (e *looseObjectEntry) Type() string {
	return e.typ
}

func (e *looseObjectEntry) Reader() io.Reader {
	return e.br
}

func (e *looseObjectEntry) Close() error {
	e.zr.Close()
	return e.f.Close()
}
