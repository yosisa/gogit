package git

import (
	"bytes"
	"io"
	"sync"
)

var bytesBufferPool = sync.Pool{
	New: func() interface{} {
		return &bytesBuffer{new(bytes.Buffer)}
	},
}

type bytesBuffer struct {
	*bytes.Buffer
}

func (b *bytesBuffer) Close() error {
	bytesBufferPool.Put(b)
	return nil
}

func acquireBytesBuffer() *bytesBuffer {
	buf := bytesBufferPool.Get().(*bytesBuffer)
	buf.Reset()
	return buf
}

func newBytesBuffer(r io.Reader) (*bytesBuffer, error) {
	buf := acquireBytesBuffer()
	if _, err := io.Copy(buf, r); err != nil {
		buf.Close()
		return nil, err
	}
	return buf, nil
}
