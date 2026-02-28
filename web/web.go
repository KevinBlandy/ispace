package web

import (
	"embed"
	_ "embed"
)

//go:embed resource
var ResourceFs embed.FS
