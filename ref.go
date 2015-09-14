package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Ref struct {
	Name   string
	SHA1   SHA1
	Commit *SHA1
}

type Refs []*Ref

func (refs Refs) merge(other []*Ref) []*Ref {
	m := refsToMap(refs)
	for _, ref := range other {
		m[ref.Name] = ref
	}
	return mapToRefs(m)
}

func (refs Refs) find(suffix string) *Ref {
	for _, ref := range refs {
		if strings.HasSuffix(ref.Name, suffix) {
			return ref
		}
	}
	return nil
}

func (r *Repository) Ref(name string) (*Ref, error) {
	if ref, err := r.looseRef(name); err == nil {
		return ref, nil
	}
	if ref := r.packedRefs.Ref(name); ref != nil {
		return ref, nil
	}
	return nil, fmt.Errorf("Ref not found: %s", name)
}

func (r *Repository) looseRef(name string) (*Ref, error) {
	f, err := os.Open(filepath.Join(r.root, name))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	b, err := buf.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return &Ref{Name: name, SHA1: SHA1FromHex(b[:len(b)-1])}, nil
}

func (r *Repository) Branches() []*Ref {
	prefix := filepath.Join("refs", "heads")
	refs := r.packedRefs.Refs(prefix)
	loose, err := r.looseRefs(prefix)
	if err != nil {
		return nil
	}
	return Refs(refs).merge(loose)
}

func (r Repository) Tags() []*Ref {
	prefix := filepath.Join("refs", "tags")
	refs := r.packedRefs.Refs(prefix)
	loose, err := r.looseRefs(prefix)
	if err != nil {
		return nil
	}
	return Refs(refs).merge(loose)
}

func (r Repository) looseRefs(path string) ([]*Ref, error) {
	path = filepath.Join(r.root, path)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var refs []*Ref
	for _, file := range files {
		name := filepath.Join(path, file.Name())
		if ref, err := r.Ref(name); err == nil {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func (r *Repository) Head() (*Ref, error) {
	f, err := os.Open(filepath.Join(r.root, "HEAD"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	b, err := buf.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(b, []byte("ref: ")) {
		return nil, ErrUnknownFormat
	}
	return r.Ref(string(b[5 : len(b)-1]))
}

type PackedRefs struct {
	Path string
	Err  error
	refs map[string]*Ref
}

func OpenPackedRefs(root string) *PackedRefs {
	return &PackedRefs{Path: filepath.Join(root, "packed-refs")}
}

func (p *PackedRefs) Ref(name string) *Ref {
	if p.refs == nil {
		if p.Err = p.Parse(); p.Err != nil {
			return nil
		}
	}
	return p.refs[name]
}

func (p *PackedRefs) Refs(prefix string) []*Ref {
	if p.refs == nil {
		if p.Err = p.Parse(); p.Err != nil {
			return nil
		}
	}
	var out []*Ref
	for _, ref := range p.refs {
		if strings.HasPrefix(ref.Name, prefix) {
			out = append(out, ref)
		}
	}
	return out
}

func (p *PackedRefs) Parse() error {
	p.refs = make(map[string]*Ref)
	f, err := os.Open(p.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	var ref *Ref
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Bytes()
		if pos := bytes.IndexByte(line, '#'); pos != -1 {
			line = line[:pos]
		}
		bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '^' {
			if ref == nil {
				return ErrUnknownFormat
			}
			commit := SHA1FromHex(line[1:])
			ref.Commit = &commit
			continue
		}
		items := bytes.Split(line, []byte{' '})
		if len(items) != 2 {
			return ErrUnknownFormat
		}
		name := string(items[1])
		ref = &Ref{Name: name, SHA1: SHA1FromHex(items[0])}
		p.refs[name] = ref
	}
	if err := scan.Err(); err != nil {
		return err
	}
	return nil
}

func refsToMap(refs []*Ref) map[string]*Ref {
	out := make(map[string]*Ref)
	for _, ref := range refs {
		out[ref.Name] = ref
	}
	return out
}

func mapToRefs(m map[string]*Ref) (refs []*Ref) {
	for _, ref := range m {
		refs = append(refs, ref)
	}
	return
}
