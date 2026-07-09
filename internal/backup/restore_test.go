package backup_test

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"slop/internal/auth"
	"slop/internal/backup"
	"slop/internal/db"
	"slop/internal/models"
)

func TestRestoreDBFile(t *testing.T) {
	root := t.TempDir()
	srcDB := filepath.Join(root, "source.db")
	destDB := filepath.Join(root, "data", "blog.db")

	database, err := db.Open(srcDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := models.SetSetting(database, "site_title", "Restored Site"); err != nil {
		t.Fatal(err)
	}
	database.Close()

	_, err = backup.Restore(srcDB, backup.RestoreOptions{
		DBPath: destDB,
		Force:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	restored, err := db.Open(destDB)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	title, err := models.GetSetting(restored, "site_title")
	if err != nil || title != "Restored Site" {
		t.Fatalf("setting: %q err=%v", title, err)
	}
}

func TestRestoreZipFile(t *testing.T) {
	root := t.TempDir()
	srcDB := filepath.Join(root, "blog.db")
	staticDir := filepath.Join(root, "static")
	uploads := filepath.Join(staticDir, "uploads")
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		t.Fatal(err)
	}

	database, err := db.Open(srcDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := models.SetSetting(database, "site_title", "Zip Restore"); err != nil {
		t.Fatal(err)
	}
	database.Close()

	imgPath := filepath.Join(uploads, "photo.jpg")
	if err := os.WriteFile(imgPath, []byte("image-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(root, "backup.zip")
	if err := writeTestZip(zipPath, srcDB, map[string][]byte{
		"uploads/photo.jpg": []byte("image-bytes"),
	}); err != nil {
		t.Fatal(err)
	}

	destDB := filepath.Join(root, "dest", "blog.db")
	destStatic := filepath.Join(root, "dest-static")
	_, err = backup.Restore(zipPath, backup.RestoreOptions{
		DBPath:    destDB,
		StaticDir: destStatic,
		Force:     true,
	})
	if err != nil {
		t.Fatal(err)
	}

	restored, err := db.Open(destDB)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	title, _ := models.GetSetting(restored, "site_title")
	if title != "Zip Restore" {
		t.Fatalf("title: %q", title)
	}

	got, err := os.ReadFile(filepath.Join(destStatic, "uploads", "photo.jpg"))
	if err != nil || string(got) != "image-bytes" {
		t.Fatalf("upload: %q err=%v", got, err)
	}
}

func TestRestoreRequiresForce(t *testing.T) {
	root := t.TempDir()
	srcDB := filepath.Join(root, "source.db")
	destDB := filepath.Join(root, "blog.db")

	database, err := db.Open(srcDB)
	if err != nil {
		t.Fatal(err)
	}
	database.Close()

	existing, err := db.Open(destDB)
	if err != nil {
		t.Fatal(err)
	}
	existing.Close()

	_, err = backup.Restore(srcDB, backup.RestoreOptions{DBPath: destDB})
	if !errors.Is(err, backup.ErrForceRequired) {
		t.Fatalf("expected ErrForceRequired, got %v", err)
	}
}

func TestRestoreDryRunNoChanges(t *testing.T) {
	root := t.TempDir()
	srcDB := filepath.Join(root, "source.db")
	destDB := filepath.Join(root, "blog.db")

	database, err := db.Open(srcDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := models.SetSetting(database, "site_title", "Before"); err != nil {
		t.Fatal(err)
	}
	database.Close()

	existing, err := db.Open(destDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := models.SetSetting(existing, "site_title", "Before"); err != nil {
		t.Fatal(err)
	}
	existing.Close()

	res, err := backup.Restore(srcDB, backup.RestoreOptions{
		DBPath: destDB,
		Force:  true,
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun {
		t.Fatal("expected dry-run result")
	}

	check, err := db.Open(destDB)
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	title, _ := models.GetSetting(check, "site_title")
	if title != "Before" {
		t.Fatalf("dry-run should not modify target, got title %q", title)
	}
}

func TestRestorePreservesAdminCredentials(t *testing.T) {
	root := t.TempDir()
	destDB := filepath.Join(root, "blog.db")
	backupDB := filepath.Join(root, "backup.db")

	currentHash, err := auth.HashPassword("current-secret")
	if err != nil {
		t.Fatal(err)
	}
	backupHash, err := auth.HashPassword("backup-secret")
	if err != nil {
		t.Fatal(err)
	}

	current, err := db.Open(destDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := models.CreateUser(current, "current-admin", currentHash, false); err != nil {
		t.Fatal(err)
	}
	if err := models.SetSetting(current, "site_title", "Current"); err != nil {
		t.Fatal(err)
	}
	current.Close()

	backupData, err := db.Open(backupDB)
	if err != nil {
		t.Fatal(err)
	}
	if err := models.CreateUser(backupData, "backup-admin", backupHash, false); err != nil {
		t.Fatal(err)
	}
	if err := models.SetSetting(backupData, "site_title", "From Backup"); err != nil {
		t.Fatal(err)
	}
	backupData.Close()

	_, err = backup.Restore(backupDB, backup.RestoreOptions{
		DBPath: destDB,
		Force:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	restored, err := db.Open(destDB)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()

	title, _ := models.GetSetting(restored, "site_title")
	if title != "From Backup" {
		t.Fatalf("content should come from backup, got title %q", title)
	}
	user, err := models.GetUserByUsername(restored, "current-admin")
	if err != nil || user == nil {
		t.Fatalf("current admin should be preserved: %v", err)
	}
	if !auth.CheckPassword(user.PasswordHash, "current-secret") {
		t.Fatal("password hash should match current admin")
	}
	if u, _ := models.GetUserByUsername(restored, "backup-admin"); u != nil {
		t.Fatal("backup admin user should not remain")
	}
}

func TestRestoreInvalidZip(t *testing.T) {
	root := t.TempDir()
	zipPath := filepath.Join(root, "bad.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	_, err = backup.Restore(zipPath, backup.RestoreOptions{
		DBPath: filepath.Join(root, "blog.db"),
		Force:  true,
	})
	if err == nil {
		t.Fatal("expected error for zip without blog.db")
	}
}

func writeTestZip(zipPath, dbPath string, extra map[string][]byte) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	if err := addFile(zw, dbPath, "blog.db"); err != nil {
		return err
	}
	for name, data := range extra {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func addFile(zw *zip.Writer, srcPath, zipName string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	w, err := zw.Create(zipName)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}
