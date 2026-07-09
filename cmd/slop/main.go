package main

import (
	"fmt"
	"os"
)

// version is injected at link time by release builds (-ldflags "-X main.version=...").
var version = "dev"

func usage() {
	fmt.Fprint(os.Stderr, `S.L.O.P. - Simple Lightweight Organizer of Posts

Usage:
  slop <command> [flags]

Commands:
  serve    Run the blog server
  passwd   Set (or create) an admin user's password
  backup   Back up the database (use --full to include uploaded files)
  import   Import content (WordPress export or SLOP backup)

Run "slop <command> -h" for command-specific flags.
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		runServe(os.Args[2:])
	case "passwd":
		runPasswd(os.Args[2:])
	case "backup":
		runBackup(os.Args[2:])
	case "import":
		runImport(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
