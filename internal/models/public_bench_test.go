package models_test

import (
	"fmt"
	"database/sql"
	"testing"
	"time"

	"slop/internal/db"
	"slop/internal/models"
)

func BenchmarkListPostsPublicPaged_10k(b *testing.B) {
	database := benchModelsDB(b, 10_000)
	defer database.Close()

	now := time.Now().UTC()
	perPage := 10
	offset := 0

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = models.CountPostsPublic(database, now)
		_, _ = models.ListPostsPublicPaged(database, perPage, offset, now)
	}
}

func BenchmarkGetPostBySlug_10k(b *testing.B) {
	database := benchModelsDB(b, 10_000)
	defer database.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = models.GetPostBySlug(database, "post-9999")
	}
}

func benchModelsDB(b *testing.B, n int) *sql.DB {
	b.Helper()
	database, err := db.Open(b.TempDir() + "/bench.db")
	if err != nil {
		b.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		slug := fmt.Sprintf("post-%d", i)
		p := &models.Post{
			Slug:         slug,
			Title:        "Post " + slug,
			BodyFormat:   "markdown",
			BodySource:   "Hello",
			RenderedHTML: "<p>Hello</p>",
			Status:       "published",
			PublishedAt:  ptrTime(now.Add(-time.Duration(i) * time.Minute)),
		}
		if i%10 == 0 {
			p.Status = "draft"
		}
		if _, err := models.CreatePost(database, p); err != nil {
			database.Close()
			b.Fatal(err)
		}
	}
	return database
}

func ptrTime(t time.Time) *time.Time { return &t }

