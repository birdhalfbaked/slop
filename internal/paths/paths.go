package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const appName = "slop"

// DataDir returns the OS-specific base directory for SLOP user data.
// Windows: %USERPROFILE%\AppData\LocalLow\slop
// macOS/Linux: $HOME/.slop
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "LocalLow", appName), nil
	}
	return filepath.Join(home, "."+appName), nil
}

// DefaultDBPath returns the default SQLite database file path ({DataDir}/db/blog.db).
func DefaultDBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "db", "blog.db"), nil
}

// DefaultStaticDir returns the default static assets directory ({DataDir}/static).
func DefaultStaticDir() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "static"), nil
}
