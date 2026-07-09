// Package slop holds the embedded web assets (admin UI and default theme) so
// they can be baked into the single binary via go:embed. The embed directive
// must live at the module root because web/ sits there.
package slop

import (
	"embed"
	"io/fs"
)

//go:embed all:web
var webFS embed.FS

// AdminFS returns the embedded admin UI templates/assets.
func AdminFS() (fs.FS, error) { return fs.Sub(webFS, "web/admin") }

// ThemeFS returns the embedded default theme fragments.
func ThemeFS() (fs.FS, error) { return fs.Sub(webFS, "web/theme/default") }
