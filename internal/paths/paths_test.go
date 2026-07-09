package paths_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"slop/internal/paths"
)

func TestDataDir(t *testing.T) {
	dir, err := paths.DataDir()
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatalf("expected absolute path, got %q", dir)
	}
	if runtime.GOOS == "windows" {
		if !strings.Contains(dir, filepath.Join("AppData", "LocalLow", "slop")) {
			t.Fatalf("windows path: %q", dir)
		}
	} else {
		if !strings.HasSuffix(dir, filepath.Join(".slop")) && !strings.HasSuffix(dir, ".slop") {
			t.Fatalf("unix path should end with .slop, got %q", dir)
		}
	}
}

func TestDefaultPaths(t *testing.T) {
	db, err := paths.DefaultDBPath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(db) != "blog.db" {
		t.Fatalf("db basename: %q", db)
	}
	if filepath.Base(filepath.Dir(db)) != "db" {
		t.Fatalf("db parent dir: %q", filepath.Dir(db))
	}

	static, err := paths.DefaultStaticDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(static) != "static" {
		t.Fatalf("static basename: %q", static)
	}
	dataDir, _ := paths.DataDir()
	if filepath.Dir(static) != dataDir {
		t.Fatalf("static should be under data dir: %q vs %q", static, dataDir)
	}
}
