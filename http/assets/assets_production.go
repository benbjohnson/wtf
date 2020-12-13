// +build production

package assets

import "embed"

//go:embed css/fontawesome.css css/theme.css
//go:embed scripts/*.js
//go:embed fonts
var fsys embed.FS
