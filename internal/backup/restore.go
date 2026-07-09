package backup

import (
	"archive/zip"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// RestoreOptions configures a backup restore.
type RestoreOptions struct {
	DBPath        string
	StaticDir     string
	Force         bool
	DryRun        bool
	SkipLockCheck bool // set when the caller has already closed the live connection
}

// Result summarizes a restore (or dry-run).
type Result struct {
	Format      string // "db" or "zip"
	Source      string
	DBPath      string
	HasStatic   bool
	UploadFiles int
	FontFiles   int
	DryRun      bool
}

var (
	// ErrForceRequired is returned when the restore target exists and --force was not passed.
	ErrForceRequired = errors.New("target exists; pass --force to overwrite")
	// ErrStopServer is returned when the target database is locked by a running server.
	ErrStopServer = errors.New("stop slop serve before restoring (database is in use)")
)

// Restore replaces the target database (and static assets for full zip backups).
func Restore(srcPath string, opts RestoreOptions) (Result, error) {
	srcPath = filepath.Clean(srcPath)
	info, err := os.Stat(srcPath)
	if err != nil {
		return Result{}, fmt.Errorf("opening backup: %w", err)
	}
	if info.IsDir() {
		return Result{}, fmt.Errorf("backup path must be a file, not a directory")
	}

	ext := strings.ToLower(filepath.Ext(srcPath))
	switch ext {
	case ".db":
		return restoreDBFile(srcPath, opts)
	case ".zip":
		return restoreZipFile(srcPath, opts)
	default:
		return Result{}, fmt.Errorf("unsupported backup format %q (expected .db or .zip)", ext)
	}
}

func restoreDBFile(srcPath string, opts RestoreOptions) (Result, error) {
	if err := validateSQLite(srcPath); err != nil {
		return Result{}, err
	}
	res := Result{
		Format: "db",
		Source: srcPath,
		DBPath: opts.DBPath,
		DryRun: opts.DryRun,
	}
	if err := checkTargets(opts, false); err != nil {
		return res, err
	}
	if opts.DryRun {
		return res, nil
	}
	if err := replaceDatabasePreservingAuth(srcPath, opts.DBPath); err != nil {
		return res, err
	}
	return res, nil
}

func restoreZipFile(srcPath string, opts RestoreOptions) (Result, error) {
	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return Result{}, fmt.Errorf("opening zip: %w", err)
	}
	defer zr.Close()

	var hasDB bool
	var uploadFiles, fontFiles int
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		switch {
		case name == "blog.db":
			hasDB = true
		case strings.HasPrefix(name, "uploads/") && !f.FileInfo().IsDir():
			uploadFiles++
		case strings.HasPrefix(name, "fonts/") && !f.FileInfo().IsDir():
			fontFiles++
		}
	}
	if !hasDB {
		return Result{}, fmt.Errorf("zip backup is missing required entry blog.db")
	}

	dbTmp, err := extractZipEntry(zr, "blog.db")
	if err != nil {
		return Result{}, err
	}
	defer os.Remove(dbTmp)

	if err := validateSQLite(dbTmp); err != nil {
		return Result{}, err
	}

	res := Result{
		Format:      "zip",
		Source:      srcPath,
		DBPath:      opts.DBPath,
		HasStatic:   uploadFiles > 0 || fontFiles > 0,
		UploadFiles: uploadFiles,
		FontFiles:   fontFiles,
		DryRun:      opts.DryRun,
	}
	if err := checkTargets(opts, true); err != nil {
		return res, err
	}
	if opts.DryRun {
		return res, nil
	}

	if err := replaceDatabasePreservingAuth(dbTmp, opts.DBPath); err != nil {
		return res, err
	}
	if err := restoreStaticFromZip(zr, opts.StaticDir); err != nil {
		return res, err
	}
	return res, nil
}

func checkTargets(opts RestoreOptions, includeStatic bool) error {
	if !opts.Force {
		if _, err := os.Stat(opts.DBPath); err == nil {
			return ErrForceRequired
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("checking database path: %w", err)
		}
		if includeStatic {
			uploads := filepath.Join(opts.StaticDir, "uploads")
			fonts := filepath.Join(opts.StaticDir, "fonts")
			if dirExists(uploads) || dirExists(fonts) {
				return ErrForceRequired
			}
		}
	}
	if !opts.DryRun && !opts.SkipLockCheck {
		if err := assertDBNotLocked(opts.DBPath); err != nil {
			return err
		}
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func assertDBNotLocked(dbPath string) error {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil
	}
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(1)", filepath.ToSlash(dbPath))
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		if isLockedErr(err) {
			return ErrStopServer
		}
		return fmt.Errorf("checking database lock: %w", err)
	}
	return nil
}

func isLockedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "locked") || strings.Contains(msg, "busy")
}

func validateSQLite(path string) error {
	dsn := fmt.Sprintf("file:%s?mode=ro", filepath.ToSlash(path))
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("opening backup database: %w", err)
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		return fmt.Errorf("backup is not a valid SQLite database: %w", err)
	}
	for _, table := range []string{"posts", "settings", "users"} {
		var name string
		err := database.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err == sql.ErrNoRows {
			return fmt.Errorf("backup database is missing table %q", table)
		}
		if err != nil {
			return fmt.Errorf("validating backup database: %w", err)
		}
	}
	return nil
}

func replaceDatabase(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating database directory: %w", err)
	}
	tmp := dest + ".restore-tmp"
	if err := copyFile(src, tmp); err != nil {
		return err
	}
	removeDBSidecars(dest)
	if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
		os.Remove(tmp)
		return fmt.Errorf("removing existing database: %w", err)
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("installing restored database: %w", err)
	}
	return nil
}

func removeDBSidecars(dbPath string) {
	for _, suffix := range []string{"-wal", "-shm"} {
		_ = os.Remove(dbPath + suffix)
	}
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("writing database: %w", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dest)
		return fmt.Errorf("copying database: %w", err)
	}
	return out.Close()
}

func extractZipEntry(zr *zip.ReadCloser, name string) (string, error) {
	var entry *zip.File
	for _, f := range zr.File {
		if filepath.ToSlash(f.Name) == name {
			entry = f
			break
		}
	}
	if entry == nil {
		return "", fmt.Errorf("zip entry %q not found", name)
	}
	rc, err := entry.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	tmp, err := os.CreateTemp("", "slop-restore-*.db")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, rc); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func restoreStaticFromZip(zr *zip.ReadCloser, staticDir string) error {
	uploads := filepath.Join(staticDir, "uploads")
	fonts := filepath.Join(staticDir, "fonts")
	if err := os.RemoveAll(uploads); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing uploads: %w", err)
	}
	if err := os.RemoveAll(fonts); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing fonts: %w", err)
	}
	if err := os.MkdirAll(uploads, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(fonts, 0o755); err != nil {
		return err
	}

	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		var destDir string
		switch {
		case strings.HasPrefix(name, "uploads/"):
			destDir = uploads
			name = strings.TrimPrefix(name, "uploads/")
		case strings.HasPrefix(name, "fonts/"):
			destDir = fonts
			name = strings.TrimPrefix(name, "fonts/")
		default:
			continue
		}
		if name == "" || strings.HasSuffix(name, "/") || f.FileInfo().IsDir() {
			continue
		}
		dest := filepath.Join(destDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := extractZipFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, rc); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
