package store

import (
	"archive/zip"
	"io"
	"io/fs"
	"ispace/common/util"
	"mime"
	"net/http"
	"path/filepath"
)

// Downloads 下载单个/多个文件
func (s *Store) Downloads(w http.ResponseWriter, paths ...string) (err error) {

	if len(paths) == 0 {
		return nil
	}

	if len(paths) == 1 {
		file := paths[0]
		stat, err := s.Stat(file)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			// 只有一个文件，且文件非目录
			file, err := s.Open(file)
			if err != nil {
				return err
			}
			defer func() {
				_ = file.Close()
			}()
			contentType := mime.TypeByExtension(filepath.Ext(stat.Name()))
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": stat.Name()}))
			_, err = io.Copy(w, file)
			return err
		}
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "download.zip"}))

	zipWriter := zip.NewWriter(w)

	defer func() {
		_ = zipWriter.Close()
	}()

	for _, path := range paths {

		stat, err := s.Stat(path)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			// 目录
			subFs, err := fs.Sub(s.FS(), path)
			if err != nil {
				return err
			}
			//if err := zipWriter.AddFS(subFs); err != nil {
			//	return err
			//}
			if err := util.AddDirFS(zipWriter, subFs, stat.Name()); err != nil {
				return err
			}
		} else {
			// 文件
			err := func() error {
				file, err := s.Open(path)
				if err != nil {
					return err
				}
				defer func() {
					_ = file.Close()
				}()
				fileWriter, err := zipWriter.Create(stat.Name()) // TODO 需要考虑到同一批文件中存在同名的文件
				if err != nil {
					return err
				}
				_, err = io.Copy(fileWriter, file)
				return err
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
