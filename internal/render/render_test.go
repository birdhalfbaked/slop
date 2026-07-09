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
