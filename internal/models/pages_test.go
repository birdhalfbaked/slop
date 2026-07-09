package models_test

import (
	"testing"

	"slop/internal/db"
	"slop/internal/models"
)

func TestStaticPageCRUD(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	p := &models.StaticPage{
		Slug:         "about",
		Title:        "About",
		BodySource:   "# Hello",
		BodyFormat:   "markdown",
		RenderedHTML: "<h1>Hello</h1>",
		Status:       "published",
	}
	id, err := models.CreateStaticPage(database, p)
	if err != nil {
		t.Fatal(err)
	}

	got, err := models.GetStaticPageBySlug(database, "about")
	if err != nil || got == nil || got.Title != "About" {
		t.Fatalf("by slug: %+v err=%v", got, err)
	}

	taken, err := models.ContentSlugExists(database, "about", 0, 0)
	if err != nil || !taken {
		t.Fatalf("slug should be taken: %v %v", taken, err)
	}
	taken, err = models.ContentSlugExists(database, "about", 0, id)
	if err != nil || taken {
		t.Fatalf("slug should be free when excluding page id: %v %v", taken, err)
	}

	p.ID = id
	p.Title = "About Us"
	if err := models.UpdateStaticPage(database, p); err != nil {
		t.Fatal(err)
	}
	got, _ = models.GetStaticPageByID(database, id)
	if got.Title != "About Us" {
		t.Fatalf("update: %q", got.Title)
	}

	pages, err := models.ListStaticPages(database)
	if err != nil || len(pages) != 1 {
		t.Fatalf("list: %d err=%v", len(pages), err)
	}

	if err := models.DeleteStaticPage(database, id); err != nil {
		t.Fatal(err)
	}
	n, _ := models.CountStaticPages(database)
	if n != 0 {
		t.Fatalf("count after delete: %d", n)
	}
}

func TestContentSlugExistsAcrossPostsAndPages(t *testing.T) {
	database, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	_, err = models.CreatePost(database, &models.Post{Slug: "shared", Title: "Post", Status: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	taken, err := models.ContentSlugExists(database, "shared", 0, 0)
	if err != nil || !taken {
		t.Fatalf("post slug taken: %v %v", taken, err)
	}

	_, err = models.CreateStaticPage(database, &models.StaticPage{Slug: "page-slug", Title: "Page", Status: "draft"})
	if err != nil {
		t.Fatal(err)
	}
	taken, err = models.ContentSlugExists(database, "page-slug", 0, 0)
	if err != nil || !taken {
		t.Fatalf("page slug taken: %v %v", taken, err)
	}
}
