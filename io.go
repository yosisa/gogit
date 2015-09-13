package git

import (
	"bufio"
	"compress/zlib"
	"io"
	"io/ioutil"
	"os"
)

type packReader struct {
	f  *os.File
	br *bufio.Reader
	zr io.ReadCloser
}

func newPackReader(f *os.File) *packReader {
	return &packReader{
		f:  f,
		br: bufio.NewReader(f),
	}
}

func (r *packReader) Read(p []byte) (int, error) {
	return r.br.Read(p)
}

func (r *packReader) ReadByte() (byte, error) {
	return r.br.ReadByte()
}

func (r *packReader) Seek(offset int64, whence int) (n int64, err error) {
	if n, err = r.f.Seek(offset, whence); err != nil {
		return
	}
	r.br.Reset(r.f)
	return
}

func (r *packReader) ZlibReader() (io.ReadCloser, error) {
	if r.zr == nil {
		zr, err := zlib.NewReader(r.br)
		if err != nil {
			return nil, err
		}
		r.zr = zr
	} else {
		if err := r.zr.(zlib.Resetter).Reset(r.br, nil); err != nil {
			return nil, err
		}
	}
	return ioutil.NopCloser(r.zr), nil
}

func (r *packReader) Close() error {
	if r.zr != nil {
		r.zr.Close()
	}
	return r.f.Close()
}

type byteReader interface {
	ReadByte() (byte, error)
}
