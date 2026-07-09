package main

import (
	"flag"
	"log"
	"net/http"

	"slop"
	"slop/internal/app"
	"slop/internal/handlers"
	"slop/internal/render"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "HTTP listen address")
	dbPath := defaultDBFlag(fs)
	staticDir := defaultStaticFlag(fs)
	_ = fs.Parse(args)

	themeFS, err := slop.ThemeFS()
	if err != nil {
		log.Fatalf("locating default theme: %v", err)
	}
	boot, err := app.Bootstrap(app.Options{DBPath: *dbPath, StaticDir: *staticDir}, themeFS)
	if err != nil {
		log.Fatalf("bootstrap: %v", err)
	}
	defer boot.DB.Close()

	if err := app.EnsureDefaultAdmin(boot.DB); err != nil {
		log.Fatalf("creating default admin: %v", err)
	}

	adminFS, err := slop.AdminFS()
	if err != nil {
		log.Fatalf("locating admin assets: %v", err)
	}

	renderer := render.New(boot.DB)
	application, err := handlers.NewApp(boot.DB, *dbPath, renderer, adminFS, boot.ThemeDefaults, *staticDir, boot.UploadDir, boot.FontDir)
	if err != nil {
		log.Fatalf("initializing app: %v", err)
	}

	log.Printf("S.L.O.P. listening on %s (admin at %s/admin)", *addr, *addr)
	if err := http.ListenAndServe(*addr, application.Routes()); err != nil {
		log.Fatal(err)
	}
}
