// Package cli is the command vigie-filezilla: it reads the site manager, converts it, writes the
// export and says what it left out.
package cli

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	_ "embed"

	"github.com/MarJC5/vigie-filezilla/internal/convert"
	"github.com/MarJC5/vigie-filezilla/internal/sitemanager"
	"github.com/MarJC5/vigie/addon/export"
	"github.com/MarJC5/vigie/addon/info"
)

// image is the add-on's image on Vigie's Add-ons page: FileZilla's logo.
//
//go:embed image.svg
var image []byte

// Run runs the command with its arguments and returns its exit status: 0 done, 1 an error, 2 a
// wrong use. version names this build in the export.
func Run(args []string, stdout, stderr io.Writer, getenv func(string) string, now func() time.Time, version string) int {
	flags := flag.NewFlagSet("vigie-filezilla", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("o", "", "write the export to this file, readable by you only (else to the standard output)")
	file := flags.String("file", "", "the site manager to read, as the last argument does")
	describe := flags.Bool(strings.TrimPrefix(info.InfoFlag, "--"), false, "print the description of this add-on for Vigie, and exit")
	vigieExport := flags.String(strings.TrimPrefix(info.ExportFlag, "--"), "", "write the export there, as -o does: Vigie gives it when it runs the add-on")
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
	if *describe {
		if err := info.Write(stdout, Info(version)); err != nil {
			fmt.Fprintln(stderr, "vigie-filezilla:", err)
			return 1
		}
		return 0
	}
	if *vigieExport != "" {
		*output = *vigieExport
	}
	if flags.NArg() > 1 || (*file != "" && flags.NArg() > 0) {
		flags.Usage()
		return 2
	}
	path := flags.Arg(0)
	if *file != "" {
		path = *file
	}
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

// Info is what the add-on says about itself to Vigie (vigie-filezilla --vigie-info).
func Info(version string) info.Info {
	both := func(en, fr string) info.Text { return info.Text{"en": en, "fr": fr} }
	return info.Info{
		Name:    "filezilla",
		Title:   both("FileZilla", "FileZilla"),
		Author:  "MarJC5",
		Release: version,
		Kind:    info.KindSource,
		Summary: both("Turns FileZilla's site manager into hosts of Vigie. Never reads the passwords.",
			"Transforme le gestionnaire de sites de FileZilla en hébergements Vigie. Ne lit jamais les mots de passe."),
		Description: both("FileZilla keeps your servers in its site manager. This add-on reads it and writes a Vigie export: "+
			"folders stay folders, each SFTP server becomes a host with its user, port, key file, comments and color. Other servers "+
			"(FTP, S3, WebDAV) are listed as left out. Vigie then finds the WordPress installs of each host.",
			"FileZilla garde vos serveurs dans son gestionnaire de sites. Cet add-on le lit et écrit un export Vigie\u00a0: "+
				"les dossiers restent des dossiers, chaque serveur SFTP devient un hébergement avec son utilisateur, son port, son "+
				"fichier de clé, ses commentaires et sa couleur. Les autres serveurs (FTP, S3, WebDAV) sont listés comme laissés de "+
				"côté. Vigie trouve ensuite les installations WordPress de chaque hébergement."),
		Details: []info.Detail{
			{Label: both("Reads", "Lit"), Value: both("FileZilla's sitemanager.xml, never the passwords it keeps",
				"Le sitemanager.xml de FileZilla, jamais les mots de passe qu'il garde")},
			{Label: both("Writes", "Écrit"), Value: both("A Vigie export", "Un export Vigie")},
			{Label: both("Network", "Réseau"), Value: both("Connects to nothing", "Ne se connecte à rien")},
		},
		Image: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(image),
		Settings: []info.Setting{{
			ID: "file", Kind: info.SettingFile, Flag: "--file",
			Label: both("Site manager", "Gestionnaire de sites"),
			Help: both("FileZilla's own when empty (~/.config/filezilla/sitemanager.xml)",
				"Celui de FileZilla s'il est vide (~/.config/filezilla/sitemanager.xml)"),
		}},
	}
}
