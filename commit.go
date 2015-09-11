package git

import (
	"bytes"
	"errors"
)

type Commit struct {
	id        SHA1
	repo      *Repository
	Tree      *Tree
	Parents   []*Commit
	Author    *User
	Committer *User
	Data      []byte
}

func newCommit(id SHA1, repo *Repository) *Commit {
	return &Commit{
		id:   id,
		repo: repo,
	}
}

func (c *Commit) SHA1() SHA1 {
	return c.id
}

func (c *Commit) Parse(data []byte) error {
	var (
		value     []byte
		parents   []*Commit
		author    *User
		committer *User
		err       error
	)

	if value, data, err = readKV(data, "tree "); err != nil {
		return err
	}
	tree := newTree(SHA1FromString(string(value)), c.repo)

	for {
		value, data, err = readKV(data, "parent ")
		if err == ErrPrefixNotMatch {
			break
		} else if err != nil {
			return err
		}
		parents = append(parents, newCommit(SHA1FromString(string(value)), c.repo))
	}

	if value, data, err = readKV(data, "author "); err != nil {
		return err
	}
	if author, err = newUser(value); err != nil {
		return err
	}

	if value, data, err = readKV(data, "committer "); err != nil {
		return err
	}
	if committer, err = newUser(value); err != nil {
		return err
	}

	c.Tree = tree
	c.Parents = parents
	c.Author = author
	c.Committer = committer
	c.Data = data[1:]
	return nil
}

func (c *Commit) Resolve() error {
	return c.repo.Resolve(c)
}

func (c *Commit) Resolved() bool {
	return c.Tree != nil
}

func (c *Commit) IsRoot() bool {
	return len(c.Parents) == 0
}

func (c *Commit) IsMerge() bool {
	return len(c.Parents) != 1
}

var ErrPrefixNotMatch = errors.New("Prefix not match")

func readKV(data []byte, prefix string) ([]byte, []byte, error) {
	if !bytes.HasPrefix(data, []byte(prefix)) {
		return nil, data, ErrPrefixNotMatch
	}
	pos := bytes.IndexByte(data, 0x0a)
	if pos == -1 {
		return nil, data, ErrUnknownFormat
	}
	return data[len(prefix):pos], data[pos+1:], nil
}
