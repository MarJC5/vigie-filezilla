package sitemanager_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/MarJC5/vigie-filezilla/internal/sitemanager"
)

func read(t *testing.T) sitemanager.Folder {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", "testdata", "sitemanager.xml"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	top, err := sitemanager.Read(f)
	if err != nil {
		t.Fatal(err)
	}
	return top
}

func TestTheFoldersAndServersOfASiteManager(t *testing.T) {
	top := read(t)
	want := sitemanager.Folder{
		Folders: []sitemanager.Folder{{
			Name: "Clients",
			Folders: []sitemanager.Folder{
				{Name: "Valais", Servers: []sitemanager.Server{
					{Name: "Atelier", Host: "ssh.example-host.ch", Port: 22, Protocol: 1, Logontype: 1, User: "atelier",
						Comments: "Contrat 2026\nrenouvelé en mars", Colour: 3},
					{Name: "Old FTP", Host: "ftp.example-host.ch", Port: 21, Protocol: 0, Logontype: 1, User: "atelier"},
				}},
				// The folder without a name is left out, as FileZilla does.
				{Name: "Web/Mobile", Servers: []sitemanager.Server{
					{Name: "Anonymous drop", Host: "sftp.example-host.ch", Port: 22, Protocol: 1, Logontype: 0},
				}},
			},
		}},
		Servers: []sitemanager.Server{
			// Named by its own text, as older versions of FileZilla wrote it.
			{Name: "VPS perso", Host: "203.0.113.24", Port: 2222, Protocol: 1, Logontype: 5, User: "deploy",
				Keyfile: "/Users/someone/.ssh/id_ed25519", Colour: 7},
			{Name: "Bucket", Host: "s3.example.test", Port: 443, Protocol: 7, Logontype: 1, User: "key"},
		},
	}
	if !reflect.DeepEqual(top, want) {
		t.Fatalf("Read() =\n%+v\nwant\n%+v", top, want)
	}
}

// The passwords of the file are kept nowhere in what is read.
func TestNoPasswordIsKept(t *testing.T) {
	read := fmt.Sprintf("%#v", read(t))
	for _, secret := range []string{"fictional-secret-42", "ZmljdGlvbmFsLXNlY3JldC00Mg==", "plain-secret-17"} {
		if strings.Contains(read, secret) {
			t.Errorf("what is read holds %q", secret)
		}
	}
}

func TestWhatIsNotASiteManager(t *testing.T) {
	for name, data := range map[string]string{
		"another root": `<?xml version="1.0"?><FileZilla2><Servers/></FileZilla2>`,
		"not XML":      `{"format": "vigie"}`,
		"empty":        ``,
	} {
		if _, err := sitemanager.Read(strings.NewReader(data)); !errors.Is(err, sitemanager.ErrNotASiteManager) {
			t.Errorf("%s: Read() = %v, want ErrNotASiteManager", name, err)
		}
	}
	// FileZilla loads nothing from a file without Servers.
	if top, err := sitemanager.Read(strings.NewReader(`<FileZilla3><Settings/></FileZilla3>`)); err != nil || len(top.Folders)+len(top.Servers) != 0 {
		t.Errorf("no Servers: %+v, %v", top, err)
	}
}

// FileZilla's settings folder: the first that exists of the XDG one, ~/.config/filezilla and
// ~/.filezilla.
func TestTheDefaultPath(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	env := func(vars map[string]string) func(string) string { return func(k string) string { return vars[k] } }
	vars := map[string]string{"HOME": home, "XDG_CONFIG_HOME": xdg}
	if got := sitemanager.DefaultPath(env(vars)); got != "" {
		t.Errorf("no folder: %q", got)
	}
	for _, dir := range []string{filepath.Join(home, ".filezilla"), filepath.Join(home, ".config", "filezilla"), filepath.Join(xdg, "filezilla")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if got, want := sitemanager.DefaultPath(env(vars)), filepath.Join(dir, "sitemanager.xml"); got != want {
			t.Errorf("DefaultPath() = %q, want %q", got, want)
		}
	}
	if got := sitemanager.DefaultPath(env(map[string]string{"HOME": "relative"})); got != "" {
		t.Errorf("a relative home: %q", got)
	}
}
