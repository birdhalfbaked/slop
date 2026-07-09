package main

import (
	"fmt"
	"os"
)

func runImport(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: slop import <subcommand> [flags]")
		fmt.Fprintln(os.Stderr, "\nSubcommands:")
		fmt.Fprintln(os.Stderr, "  wordpress  Import posts from a WordPress WXR export (.xml)")
		fmt.Fprintln(os.Stderr, "  backup     Restore from a SLOP backup (.db or .zip)")
		os.Exit(2)
	}
	switch args[0] {
	case "wordpress":
		runImportWordPress(args[1:])
	case "backup":
		runImportBackup(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown import subcommand %q\n", args[0])
		runImport(nil)
		os.Exit(2)
	}
}
