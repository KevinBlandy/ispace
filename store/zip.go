package store

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"io"
	"ispace/common"
	"ispace/common/response"
	"ispace/common/util"
	"ispace/repo/model"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ArchiveTree 压缩树
type ArchiveTree struct {
	File    string         `json:"file"`        // 完整的相对文件路径
	Title   string         `json:"title"`       // 文件名称
	Dir     bool           `json:"dir"`         // 是否是目录
	Size    int64          `json:"size,string"` // 文件大小
	Entries []*ArchiveTree `json:"entries"`     // 子项目
}

func (s *Store) archiveFile(ctx context.Context, resource *File) (*os.File, error) {
	// 读文件
	objectFile, err := s.Open(resource.Path)
	if err != nil {
		return nil, err
	}

	// 文件本身是否经过了压缩存储
	switch resource.Compression {
	case model.ObjectCompressionNone:
	case model.ObjectCompressionGzip:
		// 创建临时文件
		tmpFile, err := os.CreateTemp("", strconv.FormatInt(1, 10)+"*")
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := os.Remove(tmpFile.Name()); err != nil {
				slog.ErrorContext(ctx, "临时文件删除异常", slog.String("err", err.Error()))
			}
		}()
		//defer util.SafeClose(tmpFile)
		gzipReader, err := gzip.NewReader(objectFile)
		if err != nil {
			return nil, err
		}
		defer util.SafeClose(gzipReader)
		if _, err := io.Copy(tmpFile, gzipReader); err != nil {
			return nil, err
		}

		if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		objectFile = tmpFile
	default:
		return nil, common.NewServiceError(http.StatusBadRequest, response.Fail(response.CodeBadRequest).WithMessage("未实现的压缩格式"))
	}
	return objectFile, nil
}

// ArchiveTree 返回压缩文件的树结构
func (s *Store) ArchiveTree(path string) ([]*ArchiveTree, error) {
	objectFile, err := s.Open(path)
	if err != nil {
		return nil, err
	}
	defer util.SafeClose(objectFile)

	stat, err := objectFile.Stat()
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(objectFile, stat.Size())
	if err != nil {
		return nil, err
	}
	return s.archiveTree(zipReader), nil
}

// ServeArchiveFile 读取压缩文件中的内容
func (s *Store) ServeArchiveFile(w http.ResponseWriter, r *http.Request, path, file string) error {
	objectFile, err := s.Open(path)
	if err != nil {
		return err
	}
	defer util.SafeClose(objectFile)

	stat, err := objectFile.Stat()
	if err != nil {
		return err
	}
	zipReader, err := zip.NewReader(objectFile, stat.Size())
	if err != nil {
		return err
	}
	// 查询某个文件
	for _, f := range zipReader.File {
		if f.Name == file && !f.FileInfo().IsDir() {
			return func(f *zip.File) error {
				contentType := mime.TypeByExtension(filepath.Ext(f.FileInfo().Name()))
				if contentType == "" {
					contentType = "application/octet-stream"
				}

				w.Header().Set("Content-Type", contentType)
				w.Header().Set("Content-Length", strconv.FormatUint(f.UncompressedSize64, 10))
				download := util.BoolQuery(r.URL.Query(), "download")
				if download != nil && *download {
					w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.FileInfo().Name()}))
				}
				fileReader, err := f.Open()
				if err != nil {
					return err
				}
				defer util.SafeClose(fileReader)

				_, _ = io.Copy(w, fileReader)

				return nil
			}(f)
		}
	}
	return os.ErrNotExist
}

// archiveTree 读取 zip 文件并将其转换为树形结构
// gen by Gemini
func (s *Store) archiveTree(z *zip.Reader) []*ArchiveTree {

	var rootEntries []*ArchiveTree
	// 用于存放已创建的目录节点，Key 为完整路径 File
	nodesMap := make(map[string]*ArchiveTree)

	for _, f := range z.File {
		// 1. 标准化路径并去除末尾斜杠
		fullPath := strings.Trim(filepath.ToSlash(f.Name), "/")
		if fullPath == "" {
			continue
		}

		parts := strings.Split(fullPath, "/")
		var currentPath string

		// 2. 逐层处理路径
		for i, part := range parts {
			parentPath := currentPath
			if currentPath == "" {
				currentPath = part
			} else {
				currentPath = currentPath + "/" + part
			}

			// 判断是否为该条目自身的终点
			isLastPart := i == len(parts)-1

			// 如果该节点已存在，直接跳过进入下一层
			if _, exists := nodesMap[currentPath]; exists {
				continue
			}

			// 3. 创建新节点
			newNode := &ArchiveTree{
				File:    currentPath,
				Title:   part,
				Dir:     !isLastPart || f.FileInfo().IsDir(),
				Entries: []*ArchiveTree{},
			}

			// 只有是文件且是路径终点时，才记录大小
			if isLastPart && !f.FileInfo().IsDir() {
				newNode.Size = int64(f.UncompressedSize64)
			}

			// 4. 挂载节点
			if parentPath == "" {
				// 顶层目录
				rootEntries = append(rootEntries, newNode)
			} else {
				// 挂载到父节点
				if parentNode, ok := nodesMap[parentPath]; ok {
					parentNode.Entries = append(parentNode.Entries, newNode)
					parentNode.Dir = true // 确保父节点被标记为目录
				}
			}

			// 5. 缓存目录节点（文件节点理论上不需要缓存，但为了逻辑统一也可以放进去）
			nodesMap[currentPath] = newNode
		}
	}
	return rootEntries
}
