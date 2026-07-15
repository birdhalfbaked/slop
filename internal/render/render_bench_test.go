package render

import (
	"database/sql"
	"io"
	"strings"
	"testing"
	"time"

	"slop/internal/db"
	"slop/internal/models"
)

func BenchmarkRenderBody_MarkdownSmall(b *testing.B)  { benchRenderBody(b, 1<<10) }
func BenchmarkRenderBody_MarkdownMedium(b *testing.B) { benchRenderBody(b, 50<<10) }
func BenchmarkRenderBody_MarkdownLarge(b *testing.B)  { benchRenderBody(b, 200<<10) }

func benchRenderBody(b *testing.B, size int) {
	b.ReportAllocs()
	src := markdownFixture(size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = RenderBody("markdown", src)
	}
}

func BenchmarkRenderPage_Post(b *testing.B) {
	b.ReportAllocs()

	database := benchDB(b)
	defer database.Close()
	r := New(database)

	now := time.Now().UTC()
	post := &models.Post{
		ID:           1,
		Slug:         "bench-post",
		Title:        "Bench <em>Post</em>",
		BodyFormat:   "markdown",
		BodySource:   "# Hello\n\nSome text.\n",
		RenderedHTML: "<h1>Hello</h1><p>Some text.</p>",
		Status:       "published",
		PublishedAt:  &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.RenderPage(io.Discard, "post", post, nil, nil, nil)
	}
}

func benchDB(b *testing.B) *sql.DB {
	b.Helper()
	database, err := db.Open(b.TempDir() + "/bench.db")
	if err != nil {
		b.Fatal(err)
	}
	// Minimal settings required to render templates.
	for _, k := range []string{
		"site_title",
		"site_description",
		"base_url",
		"og_site_name",
		"twitter_handle",
		"og_default_image",
		"theme_css",
		"theme_js",
		"base_layout",
		"site_header",
		"site_footer",
		"post_header",
		"home",
		"tag",
		"post",
		"page",
		"not_found",
	} {
		if _, err := models.GetSetting(database, k); err != nil {
			database.Close()
			b.Fatal(err)
		}
	}
	return database
}

func markdownFixture(size int) string {
	seed := strings.Repeat(
		"# Heading\n\nSome paragraph text with **bold** and `code`.\n\n- a\n- b\n- c\n\n```go\npackage main\n\nfunc main() {}\n```\n\n",
		4,
	)
	if len(seed) >= size {
		return seed[:size]
	}
	var sb strings.Builder
	for sb.Len() < size {
		sb.WriteString(seed)
	}
	out := sb.String()
	if len(out) > size {
		out = out[:size]
	}
	return out
}

