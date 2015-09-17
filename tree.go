package git

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"
)

type Tree struct {
	id      SHA1
	repo    *Repository
	Entries []*TreeEntry
}

func newTree(id SHA1, repo *Repository) *Tree {
	return &Tree{
		id:   id,
		repo: repo,
	}
}

func (t *Tree) SHA1() SHA1 {
	return t.id
}

func (t *Tree) Parse(data []byte) error {
	br := bufio.NewReader(bytes.NewReader(data))
	for {
		bs, err := br.ReadBytes(' ')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return ErrUnknownFormat
		}
		mode, err := strconv.ParseInt(string(bs[:len(bs)-1]), 8, 32)
		if err != nil {
			return err
		}

		name, err := br.ReadBytes(0)
		if err != nil {
			return ErrUnknownFormat
		}
		id, err := readSHA1(br)
		if err != nil {
			return ErrUnknownFormat
		}

		t.Entries = append(t.Entries, &TreeEntry{
			Mode:   int(mode),
			Name:   string(name[:len(name)-1]),
			Object: newSparseObject(id, t.repo),
		})
	}
	return nil
}

func (t *Tree) Resolve() error {
	return t.repo.Resolve(t)
}

func (t *Tree) Resolved() bool {
	return t.Entries != nil
}

func (t *Tree) Find(path string) (*SparseObject, error) {
	path = strings.TrimLeft(path, "/")
	return t.find(strings.Split(path, "/"))
}

func (t *Tree) find(items []string) (*SparseObject, error) {
	if err := t.repo.Resolve(t); err != nil {
		return nil, err
	}
	for _, e := range t.Entries {
		if e.Name == items[0] {
			if len(items) == 1 {
				return e.Object, nil
			}
			obj, err := e.Object.Resolve()
			if err != nil {
				return nil, err
			}
			if tree, ok := obj.(*Tree); ok {
				return tree.find(items[1:])
			}
			break
		}
	}
	return nil, ErrObjectNotFound
}

type TreeEntry struct {
	Mode   int
	Name   string
	Object *SparseObject
}
