package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MarJC5/vigie-filezilla/internal/cli"
	"github.com/MarJC5/vigie/addon/addontest"
)

var sample = filepath.Join("..", "..", "testdata", "sitemanager.xml")

func run(t *testing.T, env map[string]string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cli.Run(args, &out, &errOut, func(k string) string { return env[k] }, func() time.Time { return time.Unix(0, 0) }, "test")
	return code, out.String(), errOut.String()
}

func TestTheExportGoesToTheStandardOutput(t *testing.T) {
	code, out, errOut := run(t, nil, sample)
	var f struct {
		Format string `json:"format"`
		App    string `json:"app"`
		Hosts  []any  `json:"hosts"`
	}
	if err := json.Unmarshal([]byte(out), &f); code != 0 || err != nil || f.Format != "vigie" || f.App != "vigie-filezilla test" || len(f.Hosts) != 2 {
		t.Fatalf("exit %d, %v, %+v\n%s", code, err, f, errOut)
	}
	for _, want := range []string{"3 folders, 2 hosts from SFTP servers.", "Skipped Clients/Valais/Old FTP: FTP, Vigie connects over SSH only",
		"Skipped Clients/Web-Mobile/Anonymous drop: no user"} {
		if !strings.Contains(errOut, want) {
			t.Errorf("report lacks %q:\n%s", want, errOut)
		}
	}
}

func TestTheExportFileIsTheOwnersOnly(t *testing.T) {
	file := filepath.Join(t.TempDir(), "filezilla.json")
	code, out, errOut := run(t, nil, "-o", file, sample)
	if code != 0 || out != "" || !strings.Contains(errOut, "keep it somewhere safe") {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	if info, err := os.Stat(file); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("file: %v, %v", info, err)
	}
}

// Without a path, FileZilla's own file.
func TestTheSiteManagerOfFileZillaByDefault(t *testing.T) {
	home := t.TempDir()
	if code, _, errOut := run(t, map[string]string{"HOME": home}); code != 1 || !strings.Contains(errOut, "no FileZilla settings folder") {
		t.Errorf("no FileZilla: exit %d, %s", code, errOut)
	}
	data, err := os.ReadFile(sample)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".config", "filezilla")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sitemanager.xml"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run(t, map[string]string{"HOME": home}); code != 0 {
		t.Errorf("exit %d, %s", code, errOut)
	}
}

func TestWrongUse(t *testing.T) {
	if code, _, _ := run(t, nil, "a.xml", "b.xml"); code != 2 {
		t.Errorf("two files: exit %d", code)
	}
	if code, _, errOut := run(t, nil, filepath.Join(t.TempDir(), "missing.xml")); code != 1 || errOut == "" {
		t.Errorf("a missing file: exit %d", code)
	}
	if code, _, errOut := run(t, nil, "-h"); code != 0 || !strings.Contains(errOut, "Passwords are never read") {
		t.Errorf("-h: exit %d, %s", code, errOut)
	}
}

// The add-on keeps Vigie's contract: its description, and a run as Vigie makes it.
func TestTheContractWithVigie(t *testing.T) {
	program := addontest.Build(t, "../../cmd/vigie-filezilla")
	i := addontest.Info(t, program)
	if i.Title.In("fr") != "FileZilla" || i.Image == "" || len(i.Settings) != 1 || i.Settings[0].Flag != "--file" {
		t.Errorf("description = %+v", i)
	}
	file, err := filepath.Abs(sample)
	if err != nil {
		t.Fatal(err)
	}
	f, report := addontest.Export(t, program, map[string]string{"file": file})
	if len(f.Hosts) != 2 || !strings.Contains(report, "Skipped Bucket: S3") {
		t.Errorf("export = %d hosts, report %q", len(f.Hosts), report)
	}
}
