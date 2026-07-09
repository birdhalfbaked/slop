package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"slop/internal/backup"
)

func runImportBackup(args []string) {
	fs := flag.NewFlagSet("import backup", flag.ExitOnError)
	dbPath := defaultDBFlag(fs)
	staticDir := defaultStaticFlag(fs)
	force := fs.Bool("force", false, "overwrite existing database and static files")
	dryRun := fs.Bool("dry-run", false, "validate backup and print summary without writing")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: slop import backup [flags] <backup.db|backup.zip>")
		os.Exit(2)
	}
	srcPath := fs.Arg(0)

	res, err := backup.Restore(srcPath, backup.RestoreOptions{
		DBPath:    *dbPath,
		StaticDir: *staticDir,
		Force:     *force,
		DryRun:    *dryRun,
	})
	if err != nil {
		log.Fatalf("restore failed: %v", err)
	}

	label := "Restore summary"
	if res.DryRun {
		label = "Restore summary (dry-run)"
	}
	fmt.Printf("%s\n", label)
	fmt.Printf("  Source:       %s\n", res.Source)
	fmt.Printf("  Format:       %s\n", res.Format)
	fmt.Printf("  Database:     %s\n", res.DBPath)
	if res.Format == "zip" {
		fmt.Printf("  Static files: %d uploads, %d fonts\n", res.UploadFiles, res.FontFiles)
	}
	if res.DryRun {
		fmt.Println("  No changes written.")
	} else {
		fmt.Println("  Restore complete.")
	}
}
