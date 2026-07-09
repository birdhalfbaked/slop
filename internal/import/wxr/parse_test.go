package wxr

import (
	"path/filepath"
	"runtime"
	"testing"
)

func fixturePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "sample.wxr.xml")
}

func TestParseFixture(t *testing.T) {
	export, err := ParseFile(fixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(export.Posts) != 3 {
		t.Fatalf("posts: got %d want 3", len(export.Posts))
	}
	if len(export.Attachments) != 2 {
		t.Fatalf("attachments: got %d want 2", len(export.Attachments))
	}
	if export.BaseURL != "https://example.test" {
		t.Fatalf("base URL: got %q", export.BaseURL)
	}
	if export.Attachments[100] != "https://example.test/wp-content/uploads/hero.png" {
		t.Fatalf("attachment 100: got %q", export.Attachments[100])
	}

	var chess *Post
	for i := range export.Posts {
		if export.Posts[i].Slug == "chess-is-actually-fun" {
			chess = &export.Posts[i]
			break
		}
	}
	if chess == nil {
		t.Fatal("chess-is-actually-fun not found")
	}
	if chess.Title != "Chess is actually fun" {
		t.Fatalf("title: got %q", chess.Title)
	}
	if chess.Status != "publish" {
		t.Fatalf("status: got %q", chess.Status)
	}
	if chess.PublishedAt.IsZero() {
		t.Fatal("expected published date on chess post")
	}
	if chess.PublishedAt.Year() != 2024 || chess.PublishedAt.Month() != 6 {
		t.Fatalf("published date: %v", chess.PublishedAt)
	}
	if len(chess.Tags) != 2 {
		t.Fatalf("tags: got %v", chess.Tags)
	}
	if chess.ThumbnailID != 100 {
		t.Fatalf("thumbnail: got %d", chess.ThumbnailID)
	}
	if chess.Excerpt != "A short note about chess." {
		t.Fatalf("excerpt: got %q", chess.Excerpt)
	}
}
