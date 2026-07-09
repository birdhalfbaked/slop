package models_test

import (
	"testing"
	"time"

	"slop/internal/db"
	"slop/internal/models"
)

func TestIsPubliclyVisible(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	if !models.IsPubliclyVisible(&models.Post{Status: "published", PublishedAt: &past}, now) {
		t.Fatal("past published post should be visible")
	}
	if models.IsPubliclyVisible(&models.Post{Status: "published", PublishedAt: &future}, now) {
		t.Fatal("future scheduled post should be hidden")
	}
	if models.IsPubliclyVisible(&models.Post{Status: "draft", PublishedAt: &past}, now) {
		t.Fatal("draft should be hidden")
	}
}

func TestTagSlug(t *testing.T) {
	if models.TagSlug("Chess") != "chess" {
		t.Fatalf("got %q", models.TagSlug("Chess"))
	}
	if models.TagSlug("DIY") != "diy" {
		t.Fatalf("got %q", models.TagSlug("DIY"))
	}
}

func TestListPostsByTagPaged(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Now().UTC()
	pub := now.Add(-time.Hour)
	p := &models.Post{
		Slug: "tagged", Title: "Tagged", Status: "published", PublishedAt: &pub,
		BodyFormat: "markdown", BodySource: "hi", RenderedHTML: "<p>hi</p>",
		Tags:       []string{"Chess"},
	}
	if _, err := models.CreatePost(database, p); err != nil {
		t.Fatal(err)
	}

	name, err := models.FindTagBySlug(database, "chess")
	if err != nil || name != "Chess" {
		t.Fatalf("FindTagBySlug: %q %v", name, err)
	}
	n, err := models.CountPostsByTag(database, "Chess", now)
	if err != nil || n != 1 {
		t.Fatalf("count: %d %v", n, err)
	}
	posts, err := models.ListPostsByTagPaged(database, "Chess", 10, 0, now)
	if err != nil || len(posts) != 1 {
		t.Fatalf("list: %d %v", len(posts), err)
	}
}

func TestCountPostsPublicScheduled(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	now := time.Now().UTC()
	future := now.Add(24 * time.Hour)
	p := &models.Post{
		Slug: "future", Title: "Future", Status: "published", PublishedAt: &future,
		BodyFormat: "markdown", BodySource: "x", RenderedHTML: "<p>x</p>",
	}
	if _, err := models.CreatePost(database, p); err != nil {
		t.Fatal(err)
	}
	n, err := models.CountPostsPublic(database, now)
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("future post should not count as public, got %d", n)
	}
}

func TestPostsPerPageDefault(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.Seed(database, map[string]string{}); err != nil {
		t.Fatal(err)
	}
	n, err := models.PostsPerPage(database)
	if err != nil || n != 10 {
		t.Fatalf("got %d %v", n, err)
	}
}
