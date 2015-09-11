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

		obj, err := t.repo.readObject(id, nil, true)
		if err != nil {
			return err
		}
		t.Entries = append(t.Entries, &TreeEntry{
			Mode:   int(mode),
			Name:   string(name[:len(name)-1]),
			Object: obj,
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

func (t *Tree) Find(path string) (Object, error) {
	path = strings.TrimLeft(path, "/")
	return t.find(strings.Split(path, "/"))
}

func (t *Tree) find(items []string) (Object, error) {
	if err := t.repo.Resolve(t); err != nil {
		return nil, err
	}
	for _, e := range t.Entries {
		if e.Name == items[0] {
			if len(items) == 1 {
				err := t.repo.Resolve(e.Object)
				return e.Object, err
			}
			if tree, ok := e.Object.(*Tree); ok {
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
	Object Object
}
