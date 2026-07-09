package models

import (
	"database/sql"
	"time"
)

// StaticPage is a standalone content page (not a blog post).
type StaticPage struct {
	ID           int64
	Slug         string
	Title        string
	BodySource   string
	BodyFormat   string // "markdown" | "html"
	RenderedHTML string
	Status       string // "draft" | "published"
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func scanStaticPage(row interface {
	Scan(dest ...any) error
}) (*StaticPage, error) {
	p := &StaticPage{}
	var created, updated int64
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.BodySource, &p.BodyFormat,
		&p.RenderedHTML, &p.Status, &created, &updated)
	if err != nil {
		return nil, err
	}
	p.CreatedAt = fromUnix(created)
	p.UpdatedAt = fromUnix(updated)
	return p, nil
}

const pageCols = `id, slug, title, body_source, body_format, rendered_html, status, created_at, updated_at`

func CreateStaticPage(db *sql.DB, p *StaticPage) (int64, error) {
	now := unix(time.Now())
	res, err := db.Exec(`INSERT INTO pages(slug, title, body_source, body_format, rendered_html, status, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Slug, p.Title, p.BodySource, p.BodyFormat, p.RenderedHTML, p.Status, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateStaticPage(db *sql.DB, p *StaticPage) error {
	_, err := db.Exec(`UPDATE pages SET slug=?, title=?, body_source=?, body_format=?, rendered_html=?, status=?, updated_at=?
		WHERE id=?`,
		p.Slug, p.Title, p.BodySource, p.BodyFormat, p.RenderedHTML, p.Status, unix(time.Now()), p.ID)
	return err
}

func GetStaticPageByID(db *sql.DB, id int64) (*StaticPage, error) {
	row := db.QueryRow(`SELECT `+pageCols+` FROM pages WHERE id = ?`, id)
	p, err := scanStaticPage(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func GetStaticPageBySlug(db *sql.DB, slug string) (*StaticPage, error) {
	row := db.QueryRow(`SELECT `+pageCols+` FROM pages WHERE slug = ?`, slug)
	p, err := scanStaticPage(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func ListStaticPages(db *sql.DB) ([]*StaticPage, error) {
	rows, err := db.Query(`SELECT ` + pageCols + ` FROM pages ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*StaticPage
	for rows.Next() {
		p, err := scanStaticPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func CountStaticPages(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM pages`).Scan(&n)
	return n, err
}

func DeleteStaticPage(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM pages WHERE id = ?`, id)
	return err
}
