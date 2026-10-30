// Command vigie-filezilla converts FileZilla's site manager into a Vigie export: its folders, and
// its SFTP servers as hosts. It never reads the passwords FileZilla keeps, and connects to nothing.
// Vigie imports the file it writes with a preview.
//
// Usage: vigie-filezilla [-o file] [sitemanager.xml]
package main

import (
	"os"
	"time"

	"github.com/MarJC5/vigie-filezilla/internal/cli"
)

// version is set when a release is built (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, os.Getenv, time.Now, version))
}
