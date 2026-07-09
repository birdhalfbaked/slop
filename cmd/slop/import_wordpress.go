package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"slop"
	"slop/internal/app"
	"slop/internal/import/wordpress"
)

func runImportWordPress(args []string) {
	fs := flag.NewFlagSet("import wordpress", flag.ExitOnError)
	dbPath := defaultDBFlag(fs)
	staticDir := defaultStaticFlag(fs)
	includeDrafts := fs.Bool("drafts", false, "also import draft posts")
	dryRun := fs.Bool("dry-run", false, "print summary without writing")
	onConflict := fs.String("on-conflict", "suffix", "when slug exists: skip, suffix, or overwrite")
	skipImages := fs.Bool("skip-images", false, "do not download images")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: slop import wordpress [flags] <export.xml>")
		os.Exit(2)
	}
	wxrPath := fs.Arg(0)

	themeFS, err := slop.ThemeFS()
	if err != nil {
		log.Fatalf("locating default theme: %v", err)
	}
	boot, err := app.Bootstrap(app.Options{DBPath: *dbPath, StaticDir: *staticDir}, themeFS)
	if err != nil {
		log.Fatalf("bootstrap: %v", err)
	}
	defer boot.DB.Close()

	res, err := wordpress.Import(boot.DB, boot.UploadDir, wxrPath, wordpress.Options{
		IncludeDrafts: *includeDrafts,
		DryRun:        *dryRun,
		SkipImages:    *skipImages,
		OnConflict:    *onConflict,
	})
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	label := "Import summary"
	if *dryRun {
		label = "Import summary (dry-run)"
	}
	fmt.Printf("%s\n", label)
	fmt.Printf("  Posts found:  %d\n", res.PostsFound)
	fmt.Printf("  Imported:     %d\n", res.Imported)
	fmt.Printf("  Skipped:      %d", res.Skipped)
	if res.SkippedDrafts > 0 || res.SkippedSlug > 0 {
		fmt.Print(" (")
		var parts []string
		if res.SkippedDrafts > 0 {
			parts = append(parts, fmt.Sprintf("%d drafts", res.SkippedDrafts))
		}
		if res.SkippedSlug > 0 {
			parts = append(parts, fmt.Sprintf("%d slug conflicts", res.SkippedSlug))
		}
		fmt.Print(strings.Join(parts, ", "))
		fmt.Print(")")
	}
	fmt.Println()
	fmt.Printf("  Images:       %d unique URLs\n", res.ImageURLs)
	for _, e := range res.Errors {
		fmt.Printf("  warning: %s\n", e)
	}
}
