package models

import (
	"database/sql"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var tagSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

// TagSlug converts a tag display name to a URL slug.
func TagSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = tagSlugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// IsPubliclyVisible reports whether a post should appear on the public site.
func IsPubliclyVisible(p *Post, now time.Time) bool {
	if p == nil || p.Status != "published" {
		return false
	}
	if p.PublishedAt == nil {
		return true
	}
	return !p.PublishedAt.After(now)
}

const publicPostFilter = `status = 'published' AND (published_at IS NULL OR published_at <= ?)`

// CountPostsPublic counts published posts visible as of now (excludes scheduled future posts).
func CountPostsPublic(db *sql.DB, now time.Time) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM posts WHERE `+publicPostFilter, unix(now)).Scan(&n)
	return n, err
}

// ListPostsPublicPaged returns a page of publicly visible posts, newest first.
func ListPostsPublicPaged(db *sql.DB, limit, offset int, now time.Time) ([]*Post, error) {
	q := `SELECT ` + postCols + ` FROM posts WHERE ` + publicPostFilter +
		` ORDER BY COALESCE(published_at, updated_at) DESC LIMIT ? OFFSET ?`
	rows, err := db.Query(q, unix(now), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPostList(db, rows)
}

// FindTagBySlug resolves a URL slug to the canonical tag display name.
func FindTagBySlug(db *sql.DB, slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", nil
	}
	rows, err := db.Query(`SELECT name FROM tags`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", err
		}
		if TagSlug(name) == slug {
			return name, nil
		}
	}
	return "", rows.Err()
}

// CountPostsByTag counts publicly visible posts with the given tag name.
func CountPostsByTag(db *sql.DB, tagName string, now time.Time) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM posts p
		JOIN post_tags pt ON pt.post_id = p.id
		JOIN tags t ON t.id = pt.tag_id
		WHERE t.name = ? AND `+publicPostFilter, tagName, unix(now)).Scan(&n)
	return n, err
}

// ListPostsByTagPaged returns a page of publicly visible posts for a tag.
func ListPostsByTagPaged(db *sql.DB, tagName string, limit, offset int, now time.Time) ([]*Post, error) {
	q := `SELECT p.id, p.slug, p.title, p.image, p.body_source, p.body_format, p.rendered_html,
		p.description, p.status, p.published_at, p.created_at, p.updated_at,
		p.meta_description, p.og_title, p.og_description, p.og_image
		FROM posts p
		JOIN post_tags pt ON pt.post_id = p.id
		JOIN tags t ON t.id = pt.tag_id
		WHERE t.name = ? AND ` + publicPostFilter +
		` ORDER BY COALESCE(p.published_at, p.updated_at) DESC LIMIT ? OFFSET ?`
	rows, err := db.Query(q, tagName, unix(now), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPostList(db, rows)
}

func scanPostList(db *sql.DB, rows *sql.Rows) ([]*Post, error) {
	var out []*Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, p := range out {
		tags, err := getPostTags(db, p.ID)
		if err != nil {
			return nil, err
		}
		p.Tags = tags
	}
	return out, nil
}

// PostsPerPage reads the site posts-per-page setting (default 10, max 100).
func PostsPerPage(db *sql.DB) (int, error) {
	s, err := GetSetting(db, "posts_per_page")
	if err != nil {
		return 10, err
	}
	if strings.TrimSpace(s) == "" {
		return 10, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 10, nil
	}
	if n > 100 {
		return 100, nil
	}
	return n, nil
}
