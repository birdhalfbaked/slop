package models

import (
	"database/sql"
	"strings"
	"time"
)

// Post is a blog post or draft.
type Post struct {
	ID              int64
	Slug            string
	Title           string
	Image           string
	BodySource      string
	BodyFormat      string // "markdown" | "html"
	RenderedHTML    string
	Description     string // short description, exposed to themes as .Post.Description
	Status          string // "draft" | "published"
	PublishedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	MetaDescription string
	OGTitle         string
	OGDescription   string
	OGImage         string
	Tags            []string
}

// Image is an uploaded static image.
type Image struct {
	ID         int64
	Filename   string
	Path       string
	UploadedAt time.Time
}

// Font is an installed font, either referenced via CDN or uploaded.
type Font struct {
	ID         int64
	Name       string
	Family     string
	Source     string // "cdn" | "file"
	URL        string
	Filename   string
	Weight     string
	Style      string
	UploadedAt time.Time
}

// User is an admin user.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	MustChange   bool
	CreatedAt    time.Time
}

// Session is a server-side session.
type Session struct {
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

func unix(t time.Time) int64 { return t.UTC().Unix() }

func fromUnix(v int64) time.Time { return time.Unix(v, 0).UTC() }

// ---- Settings ----

func GetSetting(db *sql.DB, key string) (string, error) {
	var v string
	err := db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func SetSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec(`INSERT INTO settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func HasSetting(db *sql.DB, key string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM settings WHERE key = ?`, key).Scan(&n)
	return n > 0, err
}

func GetAllSettings(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// ---- Users ----

// CreateUser inserts (or updates the password of) an admin user. mustChange
// marks whether the user must change their credentials on next login.
func CreateUser(db *sql.DB, username, passwordHash string, mustChange bool) error {
	_, err := db.Exec(`INSERT INTO users(username, password_hash, must_change, created_at) VALUES(?, ?, ?, ?)
		ON CONFLICT(username) DO UPDATE SET password_hash = excluded.password_hash, must_change = excluded.must_change`,
		username, passwordHash, boolToInt(mustChange), unix(time.Now()))
	return err
}

// UpdateCredentials sets a new username and password hash and clears the
// must_change flag.
func UpdateCredentials(db *sql.DB, id int64, username, passwordHash string) error {
	_, err := db.Exec(`UPDATE users SET username = ?, password_hash = ?, must_change = 0 WHERE id = ?`,
		username, passwordHash, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func scanUser(row interface{ Scan(dest ...any) error }) (*User, error) {
	u := &User{}
	var created int64
	var mustChange int
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &mustChange, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.MustChange = mustChange != 0
	u.CreatedAt = fromUnix(created)
	return u, nil
}

func GetUserByUsername(db *sql.DB, username string) (*User, error) {
	return scanUser(db.QueryRow(
		`SELECT id, username, password_hash, must_change, created_at FROM users WHERE username = ?`, username))
}

func GetUserByID(db *sql.DB, id int64) (*User, error) {
	return scanUser(db.QueryRow(
		`SELECT id, username, password_hash, must_change, created_at FROM users WHERE id = ?`, id))
}

// UsernameTaken reports whether username is used by a different user.
func UsernameTaken(db *sql.DB, username string, excludeID int64) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM users WHERE username = ? AND id != ?`, username, excludeID).Scan(&n)
	return n > 0, err
}

func CountUsers(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&n)
	return n, err
}

// ---- Sessions ----

func CreateSession(db *sql.DB, token string, userID int64, expires time.Time) error {
	_, err := db.Exec(`INSERT INTO sessions(token, user_id, expires_at, created_at) VALUES(?, ?, ?, ?)`,
		token, userID, unix(expires), unix(time.Now()))
	return err
}

func GetSession(db *sql.DB, token string) (*Session, error) {
	s := &Session{}
	var exp, created int64
	err := db.QueryRow(`SELECT token, user_id, expires_at, created_at FROM sessions WHERE token = ?`, token).
		Scan(&s.Token, &s.UserID, &exp, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.ExpiresAt = fromUnix(exp)
	s.CreatedAt = fromUnix(created)
	return s, nil
}

func DeleteSession(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func DeleteExpiredSessions(db *sql.DB) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, unix(time.Now()))
	return err
}

// ---- Posts ----

func scanPost(row interface {
	Scan(dest ...any) error
}) (*Post, error) {
	p := &Post{}
	var pub sql.NullInt64
	var created, updated int64
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Image, &p.BodySource, &p.BodyFormat,
		&p.RenderedHTML, &p.Description, &p.Status, &pub, &created, &updated,
		&p.MetaDescription, &p.OGTitle, &p.OGDescription, &p.OGImage)
	if err != nil {
		return nil, err
	}
	if pub.Valid {
		t := fromUnix(pub.Int64)
		p.PublishedAt = &t
	}
	p.CreatedAt = fromUnix(created)
	p.UpdatedAt = fromUnix(updated)
	return p, nil
}

const postCols = `id, slug, title, image, body_source, body_format, rendered_html, description, status,
	published_at, created_at, updated_at, meta_description, og_title, og_description, og_image`

func CreatePost(db *sql.DB, p *Post) (int64, error) {
	now := unix(time.Now())
	var pub any
	if p.Status == "published" {
		if p.PublishedAt == nil {
			t := time.Now().UTC()
			p.PublishedAt = &t
		}
		pub = unix(*p.PublishedAt)
	}
	res, err := db.Exec(`INSERT INTO posts(slug, title, image, body_source, body_format, rendered_html,
		description, status, published_at, created_at, updated_at, meta_description, og_title, og_description, og_image)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Slug, p.Title, p.Image, p.BodySource, p.BodyFormat, p.RenderedHTML,
		p.Description, p.Status, pub, now, now, p.MetaDescription, p.OGTitle, p.OGDescription, p.OGImage)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := setPostTags(db, id, p.Tags); err != nil {
		return 0, err
	}
	return id, nil
}

func UpdatePost(db *sql.DB, p *Post) error {
	var pub any
	if p.PublishedAt != nil {
		pub = unix(*p.PublishedAt)
	}
	_, err := db.Exec(`UPDATE posts SET slug=?, title=?, image=?, body_source=?, body_format=?,
		rendered_html=?, description=?, status=?, published_at=?, updated_at=?, meta_description=?, og_title=?,
		og_description=?, og_image=? WHERE id=?`,
		p.Slug, p.Title, p.Image, p.BodySource, p.BodyFormat, p.RenderedHTML, p.Description, p.Status, pub,
		unix(time.Now()), p.MetaDescription, p.OGTitle, p.OGDescription, p.OGImage, p.ID)
	if err != nil {
		return err
	}
	return setPostTags(db, p.ID, p.Tags)
}

func GetPostByID(db *sql.DB, id int64) (*Post, error) {
	row := db.QueryRow(`SELECT `+postCols+` FROM posts WHERE id = ?`, id)
	p, err := scanPost(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Tags, err = getPostTags(db, p.ID)
	return p, err
}

func GetPostBySlug(db *sql.DB, slug string) (*Post, error) {
	row := db.QueryRow(`SELECT `+postCols+` FROM posts WHERE slug = ?`, slug)
	p, err := scanPost(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Tags, err = getPostTags(db, p.ID)
	return p, err
}

// ListPosts returns posts. If status is non-empty, filters by it.
func ListPosts(db *sql.DB, status string) ([]*Post, error) {
	q := `SELECT ` + postCols + ` FROM posts`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY COALESCE(published_at, updated_at) DESC`
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
		if p.Tags, err = getPostTags(db, p.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// CountPosts returns the number of posts, optionally filtered by status.
func CountPosts(db *sql.DB, status string) (int, error) {
	q := `SELECT COUNT(1) FROM posts`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	var n int
	err := db.QueryRow(q, args...).Scan(&n)
	return n, err
}

// ListPostsPaged returns a page of posts (optionally filtered by status),
// newest first, with tags loaded per row.
func ListPostsPaged(db *sql.DB, status string, limit, offset int) ([]*Post, error) {
	q := `SELECT ` + postCols + ` FROM posts`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY COALESCE(published_at, updated_at) DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
		if p.Tags, err = getPostTags(db, p.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func DeletePost(db *sql.DB, id int64) error {
	if _, err := db.Exec(`DELETE FROM post_tags WHERE post_id = ?`, id); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM posts WHERE id = ?`, id)
	return err
}

// ContentSlugExists reports whether slug is used by a post or static page.
func ContentSlugExists(db *sql.DB, slug string, excludePostID, excludePageID int64) (bool, error) {
	var n int
	if err := db.QueryRow(`SELECT COUNT(1) FROM posts WHERE slug = ? AND id != ?`, slug, excludePostID).Scan(&n); err != nil {
		return false, err
	}
	if n > 0 {
		return true, nil
	}
	err := db.QueryRow(`SELECT COUNT(1) FROM pages WHERE slug = ? AND id != ?`, slug, excludePageID).Scan(&n)
	return n > 0, err
}

func SlugExists(db *sql.DB, slug string, excludeID int64) (bool, error) {
	return ContentSlugExists(db, slug, excludeID, 0)
}

// ---- Tags ----

func setPostTags(db *sql.DB, postID int64, tags []string) error {
	if _, err := db.Exec(`DELETE FROM post_tags WHERE post_id = ?`, postID); err != nil {
		return err
	}
	for _, raw := range tags {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, err := db.Exec(`INSERT INTO tags(name) VALUES(?) ON CONFLICT(name) DO NOTHING`, name); err != nil {
			return err
		}
		var tagID int64
		if err := db.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&tagID); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO post_tags(post_id, tag_id) VALUES(?, ?)
			ON CONFLICT(post_id, tag_id) DO NOTHING`, postID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func getPostTags(db *sql.DB, postID int64) ([]string, error) {
	rows, err := db.Query(`SELECT t.name FROM tags t
		JOIN post_tags pt ON pt.tag_id = t.id
		WHERE pt.post_id = ? ORDER BY t.name`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// ---- Images ----

func CreateImage(db *sql.DB, filename, path string) (int64, error) {
	res, err := db.Exec(`INSERT INTO images(filename, path, uploaded_at) VALUES(?, ?, ?)`,
		filename, path, unix(time.Now()))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListImages(db *sql.DB) ([]*Image, error) {
	rows, err := db.Query(`SELECT id, filename, path, uploaded_at FROM images ORDER BY uploaded_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Image
	for rows.Next() {
		im := &Image{}
		var up int64
		if err := rows.Scan(&im.ID, &im.Filename, &im.Path, &up); err != nil {
			return nil, err
		}
		im.UploadedAt = fromUnix(up)
		out = append(out, im)
	}
	return out, rows.Err()
}

func GetImage(db *sql.DB, id int64) (*Image, error) {
	im := &Image{}
	var up int64
	err := db.QueryRow(`SELECT id, filename, path, uploaded_at FROM images WHERE id = ?`, id).
		Scan(&im.ID, &im.Filename, &im.Path, &up)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	im.UploadedAt = fromUnix(up)
	return im, nil
}

func DeleteImage(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM images WHERE id = ?`, id)
	return err
}

// ---- Fonts ----

func CreateFont(db *sql.DB, f *Font) (int64, error) {
	res, err := db.Exec(`INSERT INTO fonts(name, family, source, url, filename, weight, style, uploaded_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)`,
		f.Name, f.Family, f.Source, f.URL, f.Filename, f.Weight, f.Style, unix(time.Now()))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func ListFonts(db *sql.DB) ([]*Font, error) {
	rows, err := db.Query(`SELECT id, name, family, source, url, filename, weight, style, uploaded_at
		FROM fonts ORDER BY uploaded_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Font
	for rows.Next() {
		f := &Font{}
		var up int64
		if err := rows.Scan(&f.ID, &f.Name, &f.Family, &f.Source, &f.URL, &f.Filename,
			&f.Weight, &f.Style, &up); err != nil {
			return nil, err
		}
		f.UploadedAt = fromUnix(up)
		out = append(out, f)
	}
	return out, rows.Err()
}

func GetFont(db *sql.DB, id int64) (*Font, error) {
	f := &Font{}
	var up int64
	err := db.QueryRow(`SELECT id, name, family, source, url, filename, weight, style, uploaded_at
		FROM fonts WHERE id = ?`, id).
		Scan(&f.ID, &f.Name, &f.Family, &f.Source, &f.URL, &f.Filename, &f.Weight, &f.Style, &up)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f.UploadedAt = fromUnix(up)
	return f, nil
}

func DeleteFont(db *sql.DB, id int64) error {
	_, err := db.Exec(`DELETE FROM fonts WHERE id = ?`, id)
	return err
}
