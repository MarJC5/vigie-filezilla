// Package cli is the command vigie-filezilla: it reads the site manager, converts it, writes the
// export and says what it left out.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/MarJC5/vigie-filezilla/internal/convert"
	"github.com/MarJC5/vigie-filezilla/internal/export"
	"github.com/MarJC5/vigie-filezilla/internal/sitemanager"
)

// Run runs the command with its arguments and returns its exit status: 0 done, 1 an error, 2 a
// wrong use. version names this build in the export.
func Run(args []string, stdout, stderr io.Writer, getenv func(string) string, now func() time.Time, version string) int {
	flags := flag.NewFlagSet("vigie-filezilla", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("o", "", "write the export to this file, readable by you only (else to the standard output)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: vigie-filezilla [-o file] [sitemanager.xml]")
		fmt.Fprintln(stderr, "")
		fmt.Fprintln(stderr, "Converts FileZilla's site manager into a Vigie export: its folders, and its SFTP servers as")
		fmt.Fprintln(stderr, "hosts. Passwords are never read. Import the file in Vigie: Settings > Data, or vigie import.")
		fmt.Fprintln(stderr, "")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 1 {
		flags.Usage()
		return 2
	}
	path := flags.Arg(0)
	if path == "" {
		if path = sitemanager.DefaultPath(getenv); path == "" {
			fmt.Fprintln(stderr, "vigie-filezilla: no FileZilla settings folder found: give the path of sitemanager.xml")
			return 1
		}
	}
	in, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(stderr, "vigie-filezilla:", err)
		return 1
	}
	defer in.Close()
	top, err := sitemanager.Read(in)
	if err != nil {
		fmt.Fprintf(stderr, "vigie-filezilla: %s: %v\n", path, err)
		return 1
	}
	f, skipped := convert.Convert(top, now(), "vigie-filezilla "+version)

	if *output == "" {
		if err := export.Write(stdout, f); err != nil {
			fmt.Fprintln(stderr, "vigie-filezilla:", err)
			return 1
		}
	} else if err := export.WriteFile(*output, f); err != nil {
		fmt.Fprintln(stderr, "vigie-filezilla:", err)
		return 1
	}
	fmt.Fprintf(stderr, "%d folders, %d hosts from SFTP servers.\n", len(f.Folders), len(f.Hosts))
	for _, s := range skipped {
		fmt.Fprintf(stderr, "Skipped %s: %s\n", s.Path, why(s))
	}
	if *output != "" {
		fmt.Fprintf(stderr, "Written to %s. It names your servers and users: keep it somewhere safe.\n", *output)
	}
	return 0
}

func why(s convert.Skipped) string {
	switch s.Reason {
	case convert.ReasonProtocol:
		return s.Protocol + ", Vigie connects over SSH only"
	case convert.ReasonNoUser:
		return "no user, Vigie connects as a named user"
	}
	return "no host"
}
