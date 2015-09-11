package git

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"io/ioutil"
)

var ErrInvalidDelta = errors.New("invalid delta")

func readDelta(r io.Reader) ([]byte, error) {
	zr, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return ioutil.ReadAll(zr)
}

func applyDelta(src io.Reader, delta []byte) (io.ReadCloser, error) {
	bs, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	data, err := applyDeltaBytes(bs, delta)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(bytes.NewReader(data)), nil
}

func applyDeltaBytes(src []byte, delta []byte) ([]byte, error) {
	br := bufio.NewReader(bytes.NewReader(delta))
	srcSize, err := deltaHeaderSize(br)
	if err != nil {
		return nil, err
	}
	if srcSize != len(src) {
		return nil, ErrInvalidDelta
	}
	dstSize, err := deltaHeaderSize(br)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			if out := buf.Bytes(); len(out) == dstSize {
				return out, nil
			}
			return nil, ErrInvalidDelta
		} else if err != nil {
			return nil, err
		}

		if b&0x80 != 0 {
			var srcOffset, srcLen int
			for i := 0; i < 4; i++ {
				if b&(1<<uint(i)) != 0 {
					b, err := br.ReadByte()
					if err != nil {
						return nil, err
					}
					srcOffset = int(b)<<uint(8*i) | srcOffset
				}
			}
			for i := 0; i < 3; i++ {
				if b&(1<<uint(4+i)) != 0 {
					b, err := br.ReadByte()
					if err != nil {
						return nil, err
					}
					srcLen = int(b)<<uint(8*i) | srcLen
				}
			}
			if srcLen == 0 {
				srcLen = 0x010000
			}
			if srcLen < 0 || srcOffset+srcLen > srcSize || srcLen > dstSize {
				return nil, ErrInvalidDelta
			}
			if _, err = buf.Write(src[srcOffset : srcOffset+srcLen]); err != nil {
				return nil, err
			}
		} else if b != 0 {
			io.Copy(buf, io.LimitReader(br, int64(b)))
		} else {
			return nil, ErrInvalidDelta
		}
	}
}

func deltaHeaderSize(br *bufio.Reader) (int, error) {
	header, err := readPackEntryHeader(br)
	if err != nil {
		return 0, err
	}
	var size int
	for i, h := range header {
		size = size | int(h.Size()<<uint(7*i))
	}
	return size, nil
}
