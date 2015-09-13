package git

import (
	"errors"
	"io"
)

var ErrInvalidDelta = errors.New("invalid delta")

func applyDelta(srcEntry objectEntry, delta *bytesBuffer) (*bytesBuffer, error) {
	defer srcEntry.Close()
	defer delta.Close()
	src, err := srcEntry.ReadAll()
	if err != nil {
		return nil, err
	}

	srcSize, err := deltaHeaderSize(delta)
	if err != nil {
		return nil, err
	}
	if srcSize != len(src) {
		return nil, ErrInvalidDelta
	}
	dstSize, err := deltaHeaderSize(delta)
	if err != nil {
		return nil, err
	}

	data := acquireBytesBuffer()
	for {
		b, err := delta.ReadByte()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		if b&0x80 != 0 {
			var srcOffset, srcLen int
			for i := 0; i < 4; i++ {
				if b&(1<<uint(i)) != 0 {
					b, err := delta.ReadByte()
					if err != nil {
						return nil, err
					}
					srcOffset = int(b)<<uint(8*i) | srcOffset
				}
			}
			for i := 0; i < 3; i++ {
				if b&(1<<uint(4+i)) != 0 {
					b, err := delta.ReadByte()
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
			if _, err = data.Write(src[srcOffset : srcOffset+srcLen]); err != nil {
				return nil, err
			}
		} else if b != 0 {
			io.Copy(data, io.LimitReader(delta, int64(b)))
		} else {
			return nil, ErrInvalidDelta
		}
	}

	if out := data.Bytes(); len(out) != dstSize {
		data.Close()
		return nil, ErrInvalidDelta
	}
	return data, nil
}

func deltaHeaderSize(br byteReader) (int, error) {
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
