package render

import (
	"strings"
	"testing"
	"time"

	"slop/internal/models"
)

func TestStripHTML(t *testing.T) {
	if got := StripHTML("<em>Hello</em> &amp; world"); got != "Hello & world" {
		t.Fatalf("got %q", got)
	}
}

func TestAbsoluteURL(t *testing.T) {
	tests := []struct {
		base, path, want string
	}{
		{"https://example.com", "/static/a.jpg", "https://example.com/static/a.jpg"},
		{"https://example.com/", "static/a.jpg", "https://example.com/static/a.jpg"},
		{"https://example.com", "https://cdn.example.com/a.jpg", "https://cdn.example.com/a.jpg"},
		{"", "/static/a.jpg", "/static/a.jpg"},
		{"https://example.com", "", ""},
	}
	for _, tt := range tests {
		if got := absoluteURL(tt.base, tt.path); got != tt.want {
			t.Fatalf("absoluteURL(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
		}
	}
}

func TestBuildMetaAbsolutizesImages(t *testing.T) {
	settings := map[string]string{
		"site_title":       "Blog",
		"site_description": "Desc",
		"base_url":         "https://example.com",
		"og_default_image": "/static/default.jpg",
		"og_site_name":     "Blog",
		"twitter_handle":   "@blog",
	}
	post := &models.Post{
		Slug:  "hello",
		Title: "Hello",
		Image: "/static/uploads/cover.jpg",
	}
	meta := string(buildMeta(settings, post, nil))
	if !strings.Contains(meta, `property="og:image" content="https://example.com/static/uploads/cover.jpg"`) {
		t.Fatalf("og:image missing absolute URL:\n%s", meta)
	}
	if !strings.Contains(meta, `name="twitter:image" content="https://example.com/static/uploads/cover.jpg"`) {
		t.Fatalf("twitter:image missing absolute URL:\n%s", meta)
	}
}

func TestToPostViewTitleFields(t *testing.T) {
	pub := time.Date(2024, 6, 2, 18, 36, 25, 0, time.UTC)
	p := &models.Post{
		Title:       "<em>Chess</em> is fun",
		PublishedAt: &pub,
		Status:      "published",
	}
	v := ToPostView(p)
	if v.Title != "<em>Chess</em> is fun" {
		t.Fatalf("title: %q", v.Title)
	}
	if !strings.Contains(string(v.TitleHTML), "<em>") {
		t.Fatalf("title html: %q", v.TitleHTML)
	}
	if v.TitlePlain != "Chess is fun" {
		t.Fatalf("title plain: %q", v.TitlePlain)
	}
	if !v.Date.Equal(pub) {
		t.Fatalf("date: %v", v.Date)
	}
}
