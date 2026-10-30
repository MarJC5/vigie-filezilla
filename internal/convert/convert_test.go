package convert_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/MarJC5/vigie-filezilla/internal/convert"
	"github.com/MarJC5/vigie-filezilla/internal/export"
	"github.com/MarJC5/vigie-filezilla/internal/sitemanager"
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

// The export matches Vigie's JSON Schema (testdata/export-v1.schema.json, a copy of Vigie's
// docs/export-v1.schema.json): the fields written are its properties, the required ones present.
func TestTheExportMatchesVigiesSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "export-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		object
		Defs map[string]object `json:"$defs"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	f, _ := convertTestdata(t)
	var buf bytes.Buffer
	if err := export.Write(&buf, f); err != nil {
		t.Fatal(err)
	}
	var written struct {
		Root    map[string]json.RawMessage
		Folders []map[string]json.RawMessage `json:"folders"`
		Hosts   []map[string]json.RawMessage `json:"hosts"`
	}
	if err := json.Unmarshal(buf.Bytes(), &written.Root); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(buf.Bytes(), &written); err != nil {
		t.Fatal(err)
	}
	check := func(name string, o object, fields map[string]json.RawMessage) {
		for key := range fields {
			if _, ok := o.Properties[key]; !ok {
				t.Errorf("%s: %q is not in Vigie's schema", name, key)
			}
		}
		for _, key := range o.Required {
			if _, ok := fields[key]; !ok {
				t.Errorf("%s: %q is required by Vigie's schema", name, key)
			}
		}
	}
	check("file", schema.object, written.Root)
	check("folder", schema.Defs["folder"], written.Folders[0])
	check("host", schema.Defs["host"], written.Hosts[0])
	if written := string(written.Root["format"]) + string(written.Root["version"]); written != `"vigie"1` {
		t.Errorf("format and version = %s", written)
	}
}

type object struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}
