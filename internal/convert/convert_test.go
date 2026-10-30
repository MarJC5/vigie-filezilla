package convert_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MarJC5/vigie-filezilla/internal/convert"
	"github.com/MarJC5/vigie-filezilla/internal/sitemanager"
	"github.com/MarJC5/vigie/addon/export"
)

var at = time.Date(2026, 10, 30, 14, 12, 0, 0, time.UTC)

func convertTestdata(t *testing.T) (export.File, []convert.Skipped) {
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
	return convert.Convert(top, at, "vigie-filezilla test")
}

func TestFoldersAndSFTPServersBecomeAVigieExport(t *testing.T) {
	f, skipped := convertTestdata(t)
	var folders, hosts []string
	for _, x := range f.Folders {
		folders = append(folders, x.Name)
	}
	for _, h := range f.Hosts {
		hosts = append(hosts, h.Name+" "+h.User+"@"+h.Hostname)
	}
	if !slices.Equal(folders, []string{"Clients", "Valais", "Web-Mobile"}) || f.Folders[1].Parent != f.Folders[0].ID {
		t.Errorf("folders = %q", folders)
	}
	if !slices.Equal(hosts, []string{"Atelier atelier@ssh.example-host.ch", "VPS perso deploy@203.0.113.24"}) {
		t.Errorf("hosts = %q", hosts)
	}
	atelier, vps := f.Hosts[0], f.Hosts[1]
	if atelier.Folder != f.Folders[1].ID || atelier.Port != 0 || atelier.Color != "blue" || atelier.Notes != "Contrat 2026\nrenouvelé en mars" ||
		atelier.KeyPath != "" || len(atelier.TrustedKeys) != 0 {
		t.Errorf("Atelier = %+v", atelier)
	}
	if vps.Folder != "" || vps.Port != 2222 || vps.KeyPath != "/Users/someone/.ssh/id_ed25519" || vps.Color != "orange" {
		t.Errorf("VPS = %+v", vps)
	}
	want := []convert.Skipped{
		{Path: "Clients/Valais/Old FTP", Reason: convert.ReasonProtocol, Protocol: "FTP"},
		{Path: "Clients/Web-Mobile/Anonymous drop", Reason: convert.ReasonNoUser},
		{Path: "Bucket", Reason: convert.ReasonProtocol, Protocol: "S3"},
	}
	if !reflect.DeepEqual(skipped, want) {
		t.Errorf("skipped =\n%+v\nwant\n%+v", skipped, want)
	}
	if f.App != "vigie-filezilla test" || !f.Exported.Equal(at) || len(f.Sites) != 0 || len(f.Ignores) != 0 {
		t.Errorf("file = %+v", f)
	}
}

// No password of FileZilla's file reaches the export, plain or encoded.
func TestTheExportHoldsNoPassword(t *testing.T) {
	f, _ := convertTestdata(t)
	var buf bytes.Buffer
	if err := export.Write(&buf, f); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fictional-secret-42", "ZmljdGlvbmFsLXNlY3JldC00Mg==", "plain-secret-17"} {
		if strings.Contains(buf.String(), secret) {
			t.Errorf("the export holds %q", secret)
		}
	}
}
