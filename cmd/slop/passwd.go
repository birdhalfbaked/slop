package main

import (
	"flag"
	"fmt"
	"log"

	"slop"
	"slop/internal/app"
)

func runPasswd(args []string) {
	fs := flag.NewFlagSet("passwd", flag.ExitOnError)
	dbPath := defaultDBFlag(fs)
	staticDir := defaultStaticFlag(fs)
	user := fs.String("user", "admin", "admin username to set the password for")
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

	pw, err := app.ReadPassword()
	if err != nil {
		log.Fatalf("reading password: %v", err)
	}
	if err := app.SetPassword(boot.DB, *user, pw); err != nil {
		log.Fatalf("setting password: %v", err)
	}
	fmt.Printf("Password updated for admin user %q.\n", *user)
}
