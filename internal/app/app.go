// Package app holds shared bootstrap and admin helpers used by the CLI
// subcommands (serve, passwd, backup).
package app

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"slop/internal/auth"
	"slop/internal/db"
	"slop/internal/models"
	"slop/internal/theme"
)

// Default credentials created on first start when no users exist. The user is
// forced to change both username and password on first login (must_change flag).
const (
	defaultAdminUser     = "admin"
	defaultAdminPassword = "admin"
)

// Options configures where data and static assets live.
type Options struct {
	DBPath    string
	StaticDir string
}

// Bootstrapped is the result of opening and preparing the application state.
type Bootstrapped struct {
	DB            *sql.DB
	ThemeDefaults map[string]string
	UploadDir     string
	FontDir       string
}

// Bootstrap ensures directories exist, opens (and migrates) the database, loads
// the default theme from themeFS, and seeds settings that are absent.
func Bootstrap(opts Options, themeFS fs.FS) (*Bootstrapped, error) {
	if err := os.MkdirAll(filepath.Dir(opts.DBPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	uploadDir := filepath.Join(opts.StaticDir, "uploads")
	fontDir := filepath.Join(opts.StaticDir, "fonts")
	for _, d := range []string{opts.StaticDir, uploadDir, fontDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("creating dir %s: %w", d, err)
		}
	}

	database, err := db.Open(opts.DBPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	themeDefaults, err := theme.LoadDefaults(themeFS)
	if err != nil {
		database.Close()
		return nil, fmt.Errorf("loading default theme: %w", err)
	}
	if err := db.Seed(database, themeDefaults); err != nil {
		database.Close()
		return nil, fmt.Errorf("seeding database: %w", err)
	}

	return &Bootstrapped{
		DB:            database,
		ThemeDefaults: themeDefaults,
		UploadDir:     uploadDir,
		FontDir:       fontDir,
	}, nil
}

// EnsureDefaultAdmin creates a default admin (forced to change credentials on
// first login) when no users exist, so no CLI step is strictly required.
func EnsureDefaultAdmin(database *sql.DB) error {
	n, err := models.CountUsers(database)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := auth.HashPassword(defaultAdminPassword)
	if err != nil {
		return err
	}
	if err := models.CreateUser(database, defaultAdminUser, hash, true); err != nil {
		return err
	}
	log.Printf("Created default admin login %q / %q (Argon2id). You will be required to change the username and password on first login.",
		defaultAdminUser, defaultAdminPassword)
	return nil
}

// SetPassword sets (or creates) an admin user's password. The must_change flag
// is cleared since the password is being set explicitly.
func SetPassword(database *sql.DB, username, password string) error {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return models.CreateUser(database, username, hash, false)
}

// ReadPassword reads a password from SLOP_ADMIN_PASSWORD or, if unset, prompts
// on stdin.
func ReadPassword() (string, error) {
	if pw := os.Getenv("SLOP_ADMIN_PASSWORD"); pw != "" {
		return pw, nil
	}
	fmt.Print("Enter new password: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	pw := strings.TrimRight(line, "\r\n")
	if pw == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	return pw, nil
}
