package constant

import (
	"os"
	"path/filepath"
)

// WorkDir 工作目录
var WorkDir string

// StorageDir 文件存储目录
var StorageDir string

// LogDir 日志目录
var LogDir string

// Database 数据库
var Database string

func init() {

	var err error

	if WorkDir, err = os.Getwd(); err != nil {
		panic(err)
	}

	StorageDir = filepath.Join(WorkDir, "storage")
	Database = filepath.Join(WorkDir, "database")
	LogDir = filepath.Join(WorkDir, "log")
}
