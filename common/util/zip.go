package util

import (
	"archive/zip"
	"io"
	"io/fs"
)

// AddDirFS 添加整个 Fs 到 zip 中，保留最顶层的目录
func AddDirFS(z *zip.Writer, f fs.FS, root string) error {

	return fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过根节点本身
		if path == "." {
			if root != "" {
				h := &zip.FileHeader{
					Name:   root + "/",
					Method: zip.Store,
				}
				h.SetMode(fs.ModeDir | 0755)
				_, err := z.CreateHeader(h)
				return err
			}
			return nil
		}

		zipPath := path
		if root != "" {
			zipPath = root + "/" + path
		}

		if d.IsDir() {
			// 目录
			info, err := d.Info()
			h, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			h.Name = zipPath + "/" // 文件必须要有 "/"
			_, err = z.CreateHeader(h)
			return err
		}

		// 文件
		file, err := f.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
		}()

		h := &zip.FileHeader{
			Name:   zipPath,
			Method: zip.Deflate,
		}

		info, err := d.Info()
		if err == nil {
			h.SetMode(info.Mode())
			h.Modified = info.ModTime()
		}

		w, err := z.CreateHeader(h)
		if err != nil {
			return err
		}

		_, err = io.Copy(w, file)
		return err
	})
}
