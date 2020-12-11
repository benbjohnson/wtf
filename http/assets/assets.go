package assets

import (
	"embed"

	"github.com/benbjohnson/hashfs"
)

//go:embed css/*.css
//go:embed scripts/*.js
//go:embed fonts
var fsys embed.FS

var FS = hashfs.NewFS(fsys)
