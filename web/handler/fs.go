package handler

import (
	"io/fs"
	"ispace/common/util"
	"ispace/config"
	"ispace/web"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

type MultiFs []http.FileSystem

func (fs MultiFs) Open(name string) (http.File, error) {
	for _, f := range fs {
		file, err := f.Open(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
		}
		return file, err
	}
	return nil, os.ErrNotExist
}

// NewFsHandler 创建新的
var NewFsHandler = func(fs ...http.FileSystem) gin.HandlerFunc {
	var f MultiFs = fs
	return func(c *gin.Context) {

		filePath := c.Request.URL.Path

		if !strings.HasPrefix(filePath, "/") {
			filePath = "/" + filePath
			c.Request.URL.Path = filePath
		}

		// 清除路径中的..等非法路径
		filePath = path.Clean(filePath)

		// 打开目标文件/或者文件夹
		file, err := f.Open(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				c.Next()
				return
			}
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}
		defer util.SafeClose(file)

		// 获取文件信息
		fileStat, err := file.Stat()
		if err != nil {
			if os.IsNotExist(err) {
				c.Next()
				return
			}
			http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
			return
		}

		// 目录的话，尝试加载下面的index.html
		if fileStat.IsDir() {

			indexFilePath := strings.TrimSuffix(filePath, "/") + "/index.html"

			indexFile, err := f.Open(indexFilePath)

			if err != nil {
				// index.html 读取异常
				if os.IsNotExist(err) {
					c.Next()
					return
				}
				http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
				return
			}

			// 有index.html文件
			defer util.SafeClose(indexFile)

			indexFileStat, err := indexFile.Stat()
			if err != nil {
				if os.IsNotExist(err) {
					c.Next()
					return
				}
				http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
				return
			}

			filePath = indexFilePath
			fileStat = indexFileStat
			file = indexFile
		}

		http.ServeContent(c.Writer, c.Request, fileStat.Name(), fileStat.ModTime(), file)

		// 中断调用链
		c.Abort()
	}
}

var DefaultFsHandler = NewFsHandler(
	http.Dir(*config.PublicDir), // 指定的公共目录优先级最高
	http.FS(util.Require(func() (fs.FS, error) {
		return fs.Sub(web.Resource, "resource/public")
	})),
)
