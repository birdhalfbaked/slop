package db

import (
	"database/sql"
	"fmt"
	"strings"

	"slop/internal/models"

	_ "modernc.org/sqlite"
)

// SEO / site default settings that are seeded (empty) if absent.
var seoDefaults = map[string]string{
	"site_title":       "S.L.O.P.",
	"site_description": "",
	"og_default_image": "",
	"og_site_name":     "",
	"twitter_handle":   "",
	"base_url":         "",
	"posts_per_page":   "10",
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	username      TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	must_change   INTEGER NOT NULL DEFAULT 0,
	created_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
	id               INTEGER PRIMARY KEY AUTOINCREMENT,
	slug             TEXT NOT NULL UNIQUE,
	title            TEXT NOT NULL DEFAULT '',
	image            TEXT NOT NULL DEFAULT '',
	body_source      TEXT NOT NULL DEFAULT '',
	body_format      TEXT NOT NULL DEFAULT 'markdown',
	rendered_html    TEXT NOT NULL DEFAULT '',
	description      TEXT NOT NULL DEFAULT '',
	status           TEXT NOT NULL DEFAULT 'draft',
	published_at     INTEGER,
	created_at       INTEGER NOT NULL,
	updated_at       INTEGER NOT NULL,
	meta_description TEXT NOT NULL DEFAULT '',
	og_title         TEXT NOT NULL DEFAULT '',
	og_description   TEXT NOT NULL DEFAULT '',
	og_image         TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS tags (
	id   INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS post_tags (
	post_id INTEGER NOT NULL,
	tag_id  INTEGER NOT NULL,
	PRIMARY KEY (post_id, tag_id)
);

CREATE TABLE IF NOT EXISTS pages (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	slug          TEXT NOT NULL UNIQUE,
	title         TEXT NOT NULL DEFAULT '',
	body_source   TEXT NOT NULL DEFAULT '',
	body_format   TEXT NOT NULL DEFAULT 'markdown',
	rendered_html TEXT NOT NULL DEFAULT '',
	status        TEXT NOT NULL DEFAULT 'draft',
	created_at    INTEGER NOT NULL,
	updated_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS images (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	filename    TEXT NOT NULL,
	path        TEXT NOT NULL,
	uploaded_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS fonts (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT NOT NULL DEFAULT '',
	family      TEXT NOT NULL DEFAULT '',
	source      TEXT NOT NULL DEFAULT 'cdn',
	url         TEXT NOT NULL DEFAULT '',
	filename    TEXT NOT NULL DEFAULT '',
	weight      TEXT NOT NULL DEFAULT '',
	style       TEXT NOT NULL DEFAULT '',
	uploaded_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL DEFAULT ''
);
`

// Open opens (creating if needed) the SQLite database and runs migrations.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", path)
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	if err := database.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(database); err != nil {
		return nil, err
	}
	return database, nil
}

func migrate(database *sql.DB) error {
	if _, err := database.Exec(schema); err != nil {
		return err
	}
	// Add columns introduced after initial release (ignore "duplicate column").
	for _, stmt := range []string{
		`ALTER TABLE users ADD COLUMN must_change INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE posts ADD COLUMN description TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err := database.Exec(stmt); err != nil && !isDuplicateColumn(err) {
			return err
		}
	}
	return nil
}

func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// Seed writes default theme fragments and SEO defaults into settings, but only
// for keys that are not already present (so admin edits are never overwritten).
func Seed(database *sql.DB, themeDefaults map[string]string) error {
	seedIfAbsent := func(key, value string) error {
		has, err := models.HasSetting(database, key)
		if err != nil {
			return err
		}
		if has {
			return nil
		}
		return models.SetSetting(database, key, value)
	}
	for key, value := range themeDefaults {
		if err := seedIfAbsent(key, value); err != nil {
			return err
		}
	}
	for key, value := range seoDefaults {
		if err := seedIfAbsent(key, value); err != nil {
			return err
		}
	}
	return nil
}
