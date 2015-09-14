package git

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"io"
	"io/ioutil"
	"os"

	"github.com/edsrzf/mmap-go"
)

type packReader interface {
	Read([]byte) (int, error)
	ReadByte() (byte, error)
	Seek(int64, int) (int64, error)
	Offset() int64
	ZlibReader() (io.ReadCloser, error)
	Close() error
}

type byteReader interface {
	ReadByte() (byte, error)
}

func newPackReader(f *os.File) packReader {
	if r, err := newMMapPackReader(f); err == nil {
		return r
	}
	return &stdPackReader{
		f:  f,
		br: bufio.NewReader(f),
	}
}

type stdPackReader struct {
	f      *os.File
	br     *bufio.Reader
	zr     io.ReadCloser
	offset int64
}

func (r *stdPackReader) Read(p []byte) (int, error) {
	return r.br.Read(p)
}

func (r *stdPackReader) ReadByte() (byte, error) {
	return r.br.ReadByte()
}

func (r *stdPackReader) Seek(offset int64, whence int) (n int64, err error) {
	if n, err = r.f.Seek(offset, whence); err != nil {
		return
	}
	r.br.Reset(r.f)
	r.offset = offset
	return
}

func (r *stdPackReader) Offset() int64 {
	return r.offset
}

func (r *stdPackReader) ZlibReader() (io.ReadCloser, error) {
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

func (r *stdPackReader) Close() error {
	if r.zr != nil {
		r.zr.Close()
	}
	return r.f.Close()
}

type mmapPackReader struct {
	f      *os.File
	mm     mmap.MMap
	size   int64
	pos    int64
	offset int64
	zr     io.ReadCloser
}

func newMMapPackReader(f *os.File) (*mmapPackReader, error) {
	mm, err := mmap.Map(f, mmap.RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return &mmapPackReader{
		f:    f,
		mm:   mm,
		size: int64(len(mm)),
	}, nil
}

func (r *mmapPackReader) Read(p []byte) (int, error) {
	if r.pos == r.size {
		return 0, io.EOF
	}
	n := copy(p, r.mm[r.pos:])
	r.pos += int64(n)
	return n, nil
}

func (r *mmapPackReader) ReadByte() (b byte, err error) {
	if r.pos == r.size {
		err = io.EOF
		return
	}
	b = r.mm[r.pos]
	r.pos++
	return
}

func (r *mmapPackReader) Seek(offset int64, whence int) (int64, error) {
	var pos int64
	switch whence {
	case os.SEEK_SET:
		pos = offset
	case os.SEEK_CUR:
		pos = r.pos + offset
	case os.SEEK_END:
		pos = r.size + offset
	}
	r.pos = pos
	r.offset = offset
	return pos, nil
}

func (r *mmapPackReader) Offset() int64 {
	return r.offset
}

func (r *mmapPackReader) ZlibReader() (io.ReadCloser, error) {
	br := bytes.NewReader(r.mm[r.pos:])
	if r.zr == nil {
		zr, err := zlib.NewReader(br)
		if err != nil {
			return nil, err
		}
		r.zr = zr
	} else {
		if err := r.zr.(zlib.Resetter).Reset(br, nil); err != nil {
			return nil, err
		}
	}
	return ioutil.NopCloser(r.zr), nil
}

func (r *mmapPackReader) Close() error {
	if r.zr != nil {
		r.zr.Close()
	}
	r.mm.Unmap()
	return r.f.Close()
}
