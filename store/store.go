package store

import (
	"ispace/common/util"
	"ispace/config"
	"ispace/repo/model"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type File struct {
	Title       string
	Compression model.ObjectCompression
	ContentType string
	Path        string
}

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
	if flag&os.O_CREATE != 0 {
		// 设置了 create 标志的时候，才尝试强制创建上级目录
		if err := s.MkdirAll(filepath.Dir(name), perm); err != nil && !os.IsExist(err) {
			return nil, err
		}
	}
	return s.Root.OpenFile(name, flag, perm)
}

// ServeContent 响应资源
func (s *Store) ServeContent(w http.ResponseWriter, r *http.Request, resource *File) error {
	// 打开资源文件
	file, err := s.Open(resource.Path)
	if err != nil {
		return err
	}
	defer util.SafeClose(file)

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if resource.ContentType != "" {
		w.Header().Set("Content-Type", resource.ContentType)
	}
	if resource.Compression != model.ObjectCompressionNone {
		w.Header().Set("Content-Encoding", string(resource.Compression))
	}
	download := util.BoolQuery(r.URL.Query(), "download")

	if download != nil && *download {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": resource.Title}))
	}

	http.ServeContent(w, r, resource.Title, stat.ModTime(), file)
	return nil
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
		panic(err)
	}
	return store
})

// DefaultChunkStore 默认的分片存储目录
var DefaultChunkStore = sync.OnceValue(func() *Store {
	store, err := New(*config.ChunkDir)
	if err != nil {
		panic(err)
	}
	return store
})
