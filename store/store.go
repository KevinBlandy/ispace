package store

import (
	"context"
	"ispace/config"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Store 文件存储实现 Store
type Store struct {
	*os.Root
}

// Destroy 彻底直接删除整个存储桶
func (s *Store) Destroy() error {
	return os.RemoveAll(s.Name())
}

// Remove 删除资源
func (s *Store) Remove(name string) error {
	return s.Root.Remove(filepath.FromSlash(name))
}

// Stat 文件信息
func (s *Store) Stat(name string) (os.FileInfo, error) {
	return s.Root.Stat(filepath.FromSlash(name))
}

// Open 读取文件
func (s *Store) Open(name string) (*os.File, error) {
	return s.Root.Open(filepath.FromSlash(name))
}

// OpenFile 创建文件，如果目录不存在会先创建目录
func (s *Store) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	name = filepath.FromSlash(name)
	if err := s.MkdirAll(filepath.Dir(name), perm); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return s.Root.OpenFile(name, flag, perm)
}

func New(dir string) (*Store, error) {
	// 直接初始化目录
	if err := os.MkdirAll(dir, os.ModePerm); err != nil && !os.IsExist(err) {
		return nil, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	return &Store{Root: root}, nil
}

// DefaultStore 全局默认的资源存储 Bucket
var DefaultStore = sync.OnceValue(func() *Store {
	store, err := New(*config.StoreDir)
	if err != nil {
		slog.ErrorContext(context.Background(), "打开存储目录异常", slog.String("err", err.Error()))
		panic(err)
	}
	return store
})
