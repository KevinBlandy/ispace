package store

import (
	"io"
	"os"
)

// Store 文件存储实现 Store
type Store struct {
	*os.Root
}

// Delete 彻底直接删除整个存储桶
func (s *Store) Delete() error {
	return os.RemoveAll(s.Name())
}

// Save 保存的库
func (s *Store) Save(reader io.Reader) error {

	// 目录打散

	file, err := s.OpenFile("", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, reader)
	return err
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
