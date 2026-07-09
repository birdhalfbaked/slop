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
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "examples", "birdhalf-baked.WordPress.2026-07-08.xml")
}

func TestParseFixture(t *testing.T) {
	export, err := ParseFile(fixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(export.Posts) != 17 {
		t.Fatalf("posts: got %d want 17", len(export.Posts))
	}
	if len(export.Attachments) != 151 {
		t.Fatalf("attachments: got %d want 151", len(export.Attachments))
	}
	if export.BaseURL != "https://birdhalfbaked.com" {
		t.Fatalf("base URL: got %q", export.BaseURL)
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
	if chess.PublishedAt.Year() != 2024 {
		t.Fatalf("published year: %d", chess.PublishedAt.Year())
	}
	if len(chess.Tags) == 0 {
		t.Fatal("expected tags on chess post")
	}
}
