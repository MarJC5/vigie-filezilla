// Package export writes Vigie's export format, version 1, as Vigie documents it (docs/data.md and
// docs/export-v1.schema.json of Vigie): the part an add-on that brings hosts needs.
package export

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// The format and the version written.
const (
	Format  = "vigie"
	Version = 1
)

// File is an export: its tree, without ignores nor history.
type File struct {
	Format   string    `json:"format"`
	Version  int       `json:"version"`
	Exported time.Time `json:"exported"`
	App      string    `json:"app"`
	Scope    string    `json:"scope"`
	Folders  []Folder  `json:"folders"`
	Hosts    []Host    `json:"hosts"`
	Sites    []Site    `json:"sites"`
	Ignores  []Ignore  `json:"ignores"`
}

// Folder is a folder; Parent is "" at the top.
type Folder struct {
	ID       string `json:"id"`
	Parent   string `json:"parent"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Notes    string `json:"notes"`
	Position int    `json:"position"`
}

// Host is an SSH account; Folder is "" at the top.
type Host struct {
	ID            string    `json:"id"`
	Folder        string    `json:"folder"`
	Name          string    `json:"name"`
	Alias         string    `json:"alias"`
	Hostname      string    `json:"hostname"`
	Port          int       `json:"port"`
	User          string    `json:"user"`
	KeyPath       string    `json:"keyPath"`
	WPCLI         string    `json:"wpCli"`
	MaxConcurrent int       `json:"maxConcurrent"`
	TrustedKeys   []HostKey `json:"trustedKeys"`
	Color         string    `json:"color"`
	Notes         string    `json:"notes"`
	Position      int       `json:"position"`
}

// HostKey is a trusted key of a server; this add-on brings none.
type HostKey struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Key  string `json:"key"`
}

// Site is a WordPress install; this add-on brings none, Vigie finds them.
type Site struct{}

// Ignore is a finding set aside; this add-on brings none.
type Ignore struct{}

// Write writes an export, indented, as Vigie does.
func Write(w io.Writer, f File) error {
	f.Format, f.Version = Format, Version
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(f)
}

// WriteFile writes an export readable by its owner only (0600), through a temporary file renamed at
// the end: it names servers and users, and an interrupted write never leaves half a file.
func WriteFile(path string, f File) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vigie-filezilla-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := Write(tmp, f); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
