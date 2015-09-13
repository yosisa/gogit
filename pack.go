package git

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var packMagic = [4]byte{'P', 'A', 'C', 'K'}

var ErrObjectNotFound = errors.New("Object not found")

type PackHeader struct {
	Magic   [4]byte
	Version uint32
	Total   uint32
}

type Pack struct {
	PackHeader
	r   *packReader
	idx *PackIndexV2
}

func OpenPack(path string) (*Pack, error) {
	path = filepath.Clean(path)
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	idx, err := OpenPackIndex(base + ".idx")
	if err != nil {
		return nil, err
	}
	f, err := os.Open(base + ".pack")
	if err != nil {
		return nil, err
	}
	pack := &Pack{
		r:   newPackReader(f),
		idx: idx,
	}
	err = pack.verify()
	return pack, err
}

func (p *Pack) verify() (err error) {
	if err = binary.Read(p.r, binary.BigEndian, &p.PackHeader); err != nil {
		return
	}
	if p.Magic != packMagic || p.Version != 2 {
		return ErrUnknownFormat
	}
	return
}

func (p *Pack) Close() error {
	return p.r.Close()
}

func (p *Pack) Object(id SHA1, repo *Repository) (Object, error) {
	entry, err := p.entry(id)
	if err != nil {
		return nil, err
	}
	obj := newObject(entry.Type(), id, repo)
	b, err := entry.ReadAll()
	if err != nil {
		return nil, err
	}
	obj.Parse(b)
	return obj, nil
}

func (p *Pack) entry(id SHA1) (*packEntry, error) {
	entry := p.idx.Entry(id)
	if entry == nil {
		return nil, ErrObjectNotFound
	}
	return p.entryAt(entry.Offset)
}

func (p *Pack) entryAt(offset int64) (*packEntry, error) {
	if _, err := p.r.Seek(offset, os.SEEK_SET); err != nil {
		return nil, err
	}

	header, err := readPackEntryHeader(p.r)
	if err != nil {
		return nil, err
	}
	size := header[0].Size0()
	typ := header[0].Type()
	for i, l := 0, len(header)-1; i < l; i++ {
		size = (header[i+1].Size() << uint(4+7*i)) | size
	}

	var pe packEntry
	switch typ {
	case packEntryCommit:
		pe.typ = "commit"
	case packEntryTree:
		pe.typ = "tree"
	case packEntryBlob:
		pe.typ = "blob"
	case packEntryTag:
		pe.typ = "tag"
	case packEntryOfsDelta:
		header, err := readPackEntryHeader(p.r)
		if err != nil {
			return nil, err
		}
		ofs := header[0].Size()
		for _, h := range header[1:] {
			ofs += 1
			ofs = (ofs << 7) + h.Size()
		}
		delta, err := p.readDelta()
		if err != nil {
			return nil, err
		}

		entry, err := p.entryAt(offset - ofs)
		if err != nil {
			return nil, err
		}
		pe.typ = entry.Type()
		if pe.buf, err = applyDelta(entry, delta); err != nil {
			return nil, err
		}
		return &pe, nil
	case packEntryRefDelta:
		id, err := readSHA1(p.r)
		if err != nil {
			return nil, err
		}
		delta, err := p.readDelta()
		if err != nil {
			return nil, err
		}

		entry, err := p.entry(id)
		if err != nil {
			return nil, err
		}
		pe.typ = entry.Type()
		if pe.buf, err = applyDelta(entry, delta); err != nil {
			return nil, err
		}
		return &pe, nil
	default:
		return nil, fmt.Errorf("Unknown pack entry type: %d", typ)
	}

	pe.pr = p.r
	return &pe, nil
}

func (p *Pack) readDelta() (*bytesBuffer, error) {
	zr, err := p.r.ZlibReader()
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return newBytesBuffer(zr)
}

type packEntryType byte

const (
	packEntryNone packEntryType = iota
	packEntryCommit
	packEntryTree
	packEntryBlob
	packEntryTag
	_
	packEntryOfsDelta
	packEntryRefDelta
)

type packEntry struct {
	typ string
	buf *bytesBuffer
	pr  *packReader
}

func (p *packEntry) Type() string {
	return p.typ
}

func (p *packEntry) ReadAll() ([]byte, error) {
	if p.buf == nil {
		zr, err := p.pr.ZlibReader()
		if err != nil {
			return nil, err
		}
		defer zr.Close()

		if p.buf, err = newBytesBuffer(zr); err != nil {
			return nil, err
		}
	}
	return p.buf.Bytes(), nil
}

func (p *packEntry) Close() (err error) {
	if p.buf != nil {
		err = p.buf.Close()
	}
	return
}

type packEntryHeader byte

func (b packEntryHeader) MSB() bool {
	return (b >> 7) == 1
}

func (b packEntryHeader) Type() packEntryType {
	return packEntryType((b >> 4) & 0x07)
}

func (b packEntryHeader) Size0() int64 {
	return int64(b & 0x0f)
}

func (b packEntryHeader) Size() int64 {
	return int64(b & 0x7f)
}

var packEntryHeaderScratch []packEntryHeader = make([]packEntryHeader, 0, 10)

func readPackEntryHeader(br byteReader) (header []packEntryHeader, err error) {
	header = packEntryHeaderScratch[:0]
	for {
		var b byte
		if b, err = br.ReadByte(); err != nil {
			return
		}
		h := packEntryHeader(b)
		header = append(header, h)
		if !h.MSB() {
			return
		}
	}
}
