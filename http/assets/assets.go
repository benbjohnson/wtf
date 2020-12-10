package assets

import "embed"

//go:embed css/*.css
//go:embed scripts/*.js
//go:embed fonts
var FS embed.FS
