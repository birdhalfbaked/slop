package backup

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Export writes a backup to w. When full is true, output is a zip archive
// (blog.db + uploads + fonts); otherwise a raw SQLite database file.
func Export(w io.Writer, database *sql.DB, staticDir string, full bool) error {
	tmp, err := os.CreateTemp("", "slop-export-*.db")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	if err := VacuumInto(database, tmpPath); err != nil {
		return fmt.Errorf("creating database copy: %w", err)
	}
	if !full {
		return copyFileToWriter(tmpPath, w)
	}
	zw := zip.NewWriter(w)
	defer zw.Close()
	if err := addFileToZip(zw, tmpPath, "blog.db"); err != nil {
		return err
	}
	if err := addDirToZip(zw, filepath.Join(staticDir, "uploads"), "uploads"); err != nil {
		return err
	}
	if err := addDirToZip(zw, filepath.Join(staticDir, "fonts"), "fonts"); err != nil {
		return err
	}
	return zw.Close()
}

// SuggestedFilename returns a timestamped backup filename.
func SuggestedFilename(full bool) string {
	ts := time.Now().Format("20060102-150405")
	if full {
		return "slop-" + ts + ".zip"
	}
	return "slop-" + ts + ".db"
}

// VacuumInto produces a consistent single-file copy of the database (safe with
// WAL mode while the database is open). The destination must not already exist.
func VacuumInto(database *sql.DB, dest string) error {
	p := strings.ReplaceAll(filepath.ToSlash(dest), "'", "''")
	_, err := database.Exec("VACUUM INTO '" + p + "'")
	return err
}

func copyFileToWriter(src string, w io.Writer) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func addFileToZip(zw *zip.Writer, srcPath, zipName string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := zw.Create(zipName)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, src)
	return err
}

func addDirToZip(zw *zip.Writer, dir, prefix string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		zipName := path.Join(prefix, filepath.ToSlash(rel))
		return addFileToZip(zw, p, zipName)
	})
}
