package constant

import (
	"os"
)

// WorkDir 工作目录
var WorkDir string

var Slash = "/"

func init() {

	var err error

	if WorkDir, err = os.Getwd(); err != nil {
		panic(err)
	}
}
