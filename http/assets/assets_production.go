// +build production

package assets

import "embed"

//go:embed css/fontawesome.css css/theme.css
//go:embed scripts/*.js
//go:embed fonts
//go:embed images
var fsys embed.FS

//go:embed index.html
var IndexHTML []byte
