package git

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Repository struct {
	Path       string
	Bare       bool
	root       string
	pack       *Pack
	packedRefs *PackedRefs
}

func Open(path string) (*Repository, error) {
	path = filepath.Clean(path)
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("Not a git repository: %s", path)
	}

	repo := &Repository{
		Path: path,
		root: path,
	}
	defer func() {
		if repo != nil {
			repo.packedRefs = OpenPackedRefs(repo.root)
		}
	}()
	if strings.HasSuffix(path, ".git") {
		repo.Bare = true
		return repo, nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.Name() == ".git" {
			repo.root = filepath.Join(repo.root, ".git")
			return repo, nil
		}
	}
	return nil, fmt.Errorf("Not a git repository: %s", path)
}

func (r *Repository) Object(id SHA1) (Object, error) {
	return r.readObject(id, nil, false)
}

func (r *Repository) Resolve(obj Object) error {
	if obj.Resolved() {
		return nil
	}
	_, err := r.readObject(obj.SHA1(), obj, false)
	return err
}

func (r *Repository) readObject(id SHA1, obj Object, headerOnly bool) (Object, error) {
	var (
		entry objectEntry
		err   error
	)
	entry, err = newLooseObjectEntry(r.root, id)
	if err != nil {
		if r.pack == nil {
			if err = r.openPack(); err != nil {
				return nil, err
			}
		}
		if entry, err = r.pack.entry(id); err != nil {
			return nil, err
		}
	}
	defer entry.Close()

	if obj == nil {
		obj = newObject(entry.Type(), id, r)
	}

	if headerOnly {
		return obj, nil
	}
	b, err := entry.ReadAll()
	if err != nil {
		return nil, err
	}
	err = obj.Parse(b)
	return obj, err
}

func (r *Repository) openPack() error {
	pattern := filepath.Join(r.root, "objects", "pack", "pack-*.pack")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	switch len(files) {
	case 0: // set empty pack
		r.pack = &Pack{idx: &PackIndexV2{}}
	case 1:
		pack, err := OpenPack(files[0])
		if err != nil {
			return err
		}
		r.pack = pack
	default:
		return errors.New("Found more than 1 pack file")
	}
	return nil
}
