package web

import (
	"embed"
	_ "embed"
)

//go:embed resource
var Resource embed.FS
