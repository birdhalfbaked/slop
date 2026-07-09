package main

import (
	"flag"
	"log"

	"slop/internal/paths"
)

func defaultDBFlag(fs *flag.FlagSet) *string {
	p, err := paths.DefaultDBPath()
	if err != nil {
		log.Fatalf("resolving default database path: %v", err)
	}
	return fs.String("db", p, "path to SQLite database file")
}

func defaultStaticFlag(fs *flag.FlagSet) *string {
	p, err := paths.DefaultStaticDir()
	if err != nil {
		log.Fatalf("resolving default static directory: %v", err)
	}
	return fs.String("static", p, "path to static assets directory")
}
