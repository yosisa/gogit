package git

type Object interface {
	SHA1() SHA1
	Parse([]byte) error
	Resolve() error
	Resolved() bool
}

func newObject(typ string, id SHA1, repo *Repository) Object {
	switch typ {
	case "blob":
		return newBlob(id, repo)
	case "tree":
		return newTree(id, repo)
	case "commit":
		return newCommit(id, repo)
	case "tag":
		return newTag(id, repo)
	}
	panic("Unknown object type: " + typ)
}

type objectEntry interface {
	Type() string
	ReadAll() ([]byte, error)
	Close() error
}
