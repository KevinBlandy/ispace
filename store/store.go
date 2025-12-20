package store

import (
	"os"
)

// Store 文件存储实现 Store
type Store struct {
	*os.Root
}

func (s *Store) Delete() error {
	return os.RemoveAll(s.Name())
}

func New(dir string) (*Store, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Store{Root: root}, nil
}

func NewFromRoot(root *os.Root) *Store {
	return &Store{Root: root}
}
