package git

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
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
	f   *os.File
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
		f:   f,
		idx: idx,
	}
	err = pack.verify()
	return pack, err
}

func (p *Pack) verify() (err error) {
	if err = binary.Read(p.f, binary.BigEndian, &p.PackHeader); err != nil {
		return
	}
	if p.Magic != packMagic || p.Version != 2 {
		return ErrUnknownFormat
	}
	return
}

func (p *Pack) Close() error {
	return p.f.Close()
}

func (p *Pack) Object(id SHA1, repo *Repository) (Object, error) {
	entry, err := p.entry(id)
	if err != nil {
		return nil, err
	}
	obj := newObject(entry.Type(), id, repo)

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, entry.Reader()); err != nil {
		return nil, err
	}
	obj.Parse(buf.Bytes())
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
	if _, err := p.f.Seek(offset, os.SEEK_SET); err != nil {
		return nil, err
	}

	br := bufio.NewReader(p.f)
	header, err := readPackEntryHeader(br)
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
		header, err := readPackEntryHeader(br)
		if err != nil {
			return nil, err
		}
		ofs := header[0].Size()
		for _, h := range header[1:] {
			ofs += 1
			ofs = (ofs << 7) + h.Size()
		}
		delta, err := readDelta(br)
		if err != nil {
			return nil, err
		}
		entry, err := p.entryAt(offset - ofs)
		if err != nil {
			return nil, err
		}
		pe.typ = entry.Type()
		if pe.r, err = applyDelta(entry.Reader(), delta); err != nil {
			return nil, err
		}
		return &pe, nil
	case packEntryRefDelta:
		id, err := readSHA1(br)
		if err != nil {
			return nil, err
		}
		delta, err := readDelta(br)
		if err != nil {
			return nil, err
		}
		entry, err := p.entry(id)
		if err != nil {
			return nil, err
		}
		pe.typ = entry.Type()
		if pe.r, err = applyDelta(entry.Reader(), delta); err != nil {
			return nil, err
		}
		return &pe, nil
	default:
		return nil, fmt.Errorf("Unknown pack entry type: %d", typ)
	}

	if pe.r, err = zlib.NewReader(br); err != nil {
		return nil, err
	}
	return &pe, nil
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
	r   io.ReadCloser
}

func (p *packEntry) Type() string {
	return p.typ
}

func (p *packEntry) Reader() io.Reader {
	return p.r
}

func (p *packEntry) Close() error {
	return p.r.Close()
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

func readPackEntryHeader(br *bufio.Reader) (header []packEntryHeader, err error) {
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
