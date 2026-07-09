package backup_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"slop/internal/backup"
	"slop/internal/db"
	"slop/internal/models"
)

func TestExportDatabase(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "blog.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := models.SetSetting(database, "site_title", "Export Test"); err != nil {
		t.Fatal(err)
	}
	database.Close()

	database, err = db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	var buf bytes.Buffer
	if err := backup.Export(&buf, database, root, false); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("empty export")
	}

	tmp := filepath.Join(root, "out.db")
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := db.Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	title, _ := models.GetSetting(out, "site_title")
	if title != "Export Test" {
		t.Fatalf("title: %q", title)
	}
}

func TestExportFullZip(t *testing.T) {
	root := t.TempDir()
	staticDir := filepath.Join(root, "static")
	uploads := filepath.Join(staticDir, "uploads")
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "a.jpg"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(root, "blog.db")
	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	var buf bytes.Buffer
	if err := backup.Export(&buf, database, staticDir, true); err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var hasDB, hasUpload bool
	for _, f := range zr.File {
		switch f.Name {
		case "blog.db":
			hasDB = true
		case "uploads/a.jpg":
			hasUpload = true
		}
	}
	if !hasDB || !hasUpload {
		t.Fatalf("zip contents: db=%v upload=%v", hasDB, hasUpload)
	}
}

func TestSuggestedFilename(t *testing.T) {
	if !strings.HasSuffix(backup.SuggestedFilename(false), ".db") {
		t.Fatal("expected .db")
	}
	if !strings.HasSuffix(backup.SuggestedFilename(true), ".zip") {
		t.Fatal("expected .zip")
	}
}
