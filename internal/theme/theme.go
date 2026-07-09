package theme

import (
	"fmt"
	"io/fs"
)

// Fragment describes a themeable template/asset stored in settings.
type Fragment struct {
	Key      string // settings key
	File     string // filename inside the embedded default theme dir
	Label    string // human label for the admin UI
	Language string // "html" | "css" | "js" (for editor hints)
}

// Fragments are all editable theme fragments, in display order.
var Fragments = []Fragment{
	{Key: "base_layout", File: "base_layout.html", Label: "Base Layout", Language: "html"},
	{Key: "site_header", File: "site_header.html", Label: "Site Header", Language: "html"},
	{Key: "site_footer", File: "site_footer.html", Label: "Site Footer", Language: "html"},
	{Key: "post_header", File: "post_header.html", Label: "Post Header", Language: "html"},
	{Key: "home", File: "home.html", Label: "Home / Post List", Language: "html"},
	{Key: "tag", File: "tag.html", Label: "Tag Archive", Language: "html"},
	{Key: "post", File: "post.html", Label: "Single Post", Language: "html"},
	{Key: "page", File: "page.html", Label: "Static Page", Language: "html"},
	{Key: "not_found", File: "not_found.html", Label: "404 Not Found", Language: "html"},
	{Key: "theme_css", File: "theme.css", Label: "Theme CSS", Language: "css"},
	{Key: "theme_js", File: "theme.js", Label: "Theme JS", Language: "js"},
}

// FragmentByKey returns the fragment definition for a settings key.
func FragmentByKey(key string) (Fragment, bool) {
	for _, f := range Fragments {
		if f.Key == key {
			return f, true
		}
	}
	return Fragment{}, false
}

// IsFragmentKey reports whether key is a known theme fragment.
func IsFragmentKey(key string) bool {
	_, ok := FragmentByKey(key)
	return ok
}

// LoadDefaults reads all default fragment contents from the embedded theme FS.
// defaultFS should be rooted at the default theme directory (web/theme/default).
func LoadDefaults(defaultFS fs.FS) (map[string]string, error) {
	out := map[string]string{}
	for _, f := range Fragments {
		b, err := fs.ReadFile(defaultFS, f.File)
		if err != nil {
			return nil, fmt.Errorf("reading default theme %q: %w", f.File, err)
		}
		out[f.Key] = string(b)
	}
	return out, nil
}
