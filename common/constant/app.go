package constant

import (
	"os"
	"time"
)

// WorkDir 工作目录
var WorkDir string

var Slash = "/"

// Location 默认的时区
var Location *time.Location

func init() {

	var err error

	if WorkDir, err = os.Getwd(); err != nil {
		panic(err)

	}

	Location, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
}
