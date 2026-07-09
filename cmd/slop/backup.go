package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"slop/internal/backup"
	"slop/internal/db"
)

func runBackup(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	dbPath := defaultDBFlag(fs)
	staticDir := defaultStaticFlag(fs)
	out := fs.String("out", "backups", "output directory for backups")
	full := fs.Bool("full", false, "include uploaded images/fonts in a .zip archive")
	_ = fs.Parse(args)

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}
	ts := time.Now().Format("20060102-150405")

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("opening database: %v", err)
	}
	defer database.Close()

	if !*full {
		dest := filepath.Join(*out, "slop-"+ts+".db")
		f, err := os.Create(dest)
		if err != nil {
			log.Fatalf("backup failed: %v", err)
		}
		if err := backup.Export(f, database, *staticDir, false); err != nil {
			f.Close()
			log.Fatalf("backup failed: %v", err)
		}
		f.Close()
		fmt.Printf("Database backup written to %s\n", dest)
		return
	}

	zipPath := filepath.Join(*out, "slop-"+ts+".zip")
	f, err := os.Create(zipPath)
	if err != nil {
		log.Fatalf("backup failed: %v", err)
	}
	if err := backup.Export(f, database, *staticDir, true); err != nil {
		f.Close()
		log.Fatalf("backup failed: %v", err)
	}
	f.Close()
	fmt.Printf("Full backup written to %s\n", zipPath)
}
