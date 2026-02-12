package store

import (
	"archive/zip"
	"compress/gzip"
	"errors"
	"io"
	"ispace/common/util"
	"ispace/repo/model"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
)

// DownloadTree 下载树
type DownloadTree struct {
	Id          int64                   `json:"-"`           // 记录 ID
	ParentId    int64                   `json:"-"`           // 父级 ID
	Path        string                  `json:"path"`        // 物理路径
	Title       string                  `json:"title"`       // 标题
	Dir         bool                    `json:"dir"`         // 是否是目录
	Compression model.ObjectCompression `json:"compression"` // 压缩
	Entries     []*DownloadTree         `json:"entries"`     // 子项目列表
}

// Downloads 下载单个/多个文件
func (s *Store) Downloads(w http.ResponseWriter, files ...*DownloadTree) (err error) {

	if len(files) == 0 {
		//w.WriteHeader(http.StatusNoContent)
		http.NotFound(w, nil)
		return nil
	}

	// 只有一个下载目标，且是文件
	if len(files) == 1 {

		singleFile := files[0]

		if !singleFile.Dir {
			file, err := s.Open(singleFile.Path)
			if err != nil {
				return err
			}

			stat, err := file.Stat()
			if err != nil {
				return err
			}

			defer func() {
				_ = file.Close()
			}()

			contentType := mime.TypeByExtension(filepath.Ext(singleFile.Title))
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			if singleFile.Compression != model.ObjectCompressionNone {
				w.Header().Set("Content-Encoding", string(singleFile.Compression))
			}

			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": singleFile.Title}))
			_, err = io.Copy(w, file)
			return err
		}
	}

	// 文件存在多个的时候，下载为 zip 格式
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment",
		map[string]string{"filename": "ispace-download.zip"}),
	)

	zipWriter := zip.NewWriter(w)

	defer func() {
		_ = zipWriter.Close()
	}()

	var handler func(*zip.Writer, string, *DownloadTree) error

	handler = func(zipWriter *zip.Writer, treePath string, tree *DownloadTree) error {

		completePath := path.Join(treePath, tree.Title)

		if tree.Dir {
			// 创建目录
			if _, err := zipWriter.CreateHeader(&zip.FileHeader{
				Name:   completePath + "/", // "/" 结尾表示目录
				Method: zip.Store,          // 目录不压缩
			}); err != nil {
				return err
			}
			// 递归
			for _, entry := range tree.Entries {
				if err := handler(zipWriter, completePath, entry); err != nil {
					return err
				}
			}
			return nil
		}

		// 写入文件
		file, err := s.Open(tree.Path)
		if err != nil {
			return err
		}

		var fileReader io.ReadCloser = file

		defer util.SafeClose(fileReader)

		// 压缩状态判断
		switch tree.Compression {
		case model.ObjectCompressionNone:
		case model.ObjectCompressionGzip:
			if fileReader, err = gzip.NewReader(file); err != nil {
				return err
			}
		default:
			return errors.New("未实现的压缩格式")
		}

		fileWriter, err := zipWriter.Create(completePath)
		if err != nil {
			return err
		}
		_, err = io.Copy(fileWriter, fileReader)
		return err
	}

	// 递归写入所有的资源
	for _, file := range files {
		if err := handler(zipWriter, "", file); err != nil {
			return err
		}
	}
	return nil
}
