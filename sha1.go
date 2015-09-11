package git

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
)

type SHA1 [20]byte

func (b SHA1) String() string {
	return hex.EncodeToString(b[:])
}

func (b SHA1) Compare(other SHA1) int {
	return bytes.Compare(b[:], other[:])
}

func NewSHA1(s string) (sha SHA1, err error) {
	var b []byte
	if b, err = hex.DecodeString(s); err == nil {
		copy(sha[:], b)
	}
	return
}

func SHA1FromString(s string) SHA1 {
	sha, err := NewSHA1(s)
	if err != nil {
		panic(err)
	}
	return sha
}

func readSHA1(r io.Reader) (sha SHA1, err error) {
	err = binary.Read(r, binary.BigEndian, &sha)
	return
}
