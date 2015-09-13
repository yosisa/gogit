package git

type Blob struct {
	id   SHA1
	repo *Repository
	Data []byte
}

func newBlob(id SHA1, repo *Repository) *Blob {
	return &Blob{
		id:   id,
		repo: repo,
	}
}

func (b *Blob) SHA1() SHA1 {
	return b.id
}

func (b *Blob) Parse(data []byte) error {
	b.Data = cloneBytes(data)
	return nil
}

func (b *Blob) Resolve() error {
	return b.repo.Resolve(b)
}

func (b *Blob) Resolved() bool {
	return b.Data != nil
}

func cloneBytes(b []byte) []byte {
	n := len(b)
	dst := make([]byte, n, n)
	copy(dst, b)
	return dst
}
