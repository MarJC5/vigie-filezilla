// Package convert turns FileZilla's site manager into a Vigie export: its folders, and its SFTP
// servers as hosts. Vigie reaches its sites over SSH only, so the other servers are listed as
// skipped; it finds the WordPress installs of a host itself, so no site is brought.
package convert

import (
	"strings"
	"time"
	"uuid"

	"github.com/MarJC5/vigie-filezilla/internal/sitemanager"
	"github.com/MarJC5/vigie/addon/export"
)

// Reason tells why a server of FileZilla is not brought.
type Reason string

const (
	ReasonProtocol Reason = "protocol" // not SFTP
	ReasonNoUser   Reason = "no-user"  // Vigie connects as a named user
	ReasonNoHost   Reason = "no-host"
)

// Skipped is a server left out, with its path in the site manager and why.
type Skipped struct {
	Path     string
	Reason   Reason
	Protocol string // for ReasonProtocol: its name
}

// Colors of the export, in FileZilla's order (src/commonui/site_color.h), which is Vigie's.
var colors = []string{"none", "red", "green", "blue", "yellow", "cyan", "magenta", "orange"}

// The names of FileZilla's protocols (src/include/server.h), for the servers skipped.
var protocols = []string{"FTP", "SFTP", "HTTP", "FTPS", "FTPES", "HTTPS", "FTP (insecure)", "S3", "Storj", "WebDAV",
	"Azure File", "Azure Blob", "Swift", "Google Cloud", "Google Drive", "Dropbox", "OneDrive", "B2", "Box",
	"WebDAV (insecure)", "Rackspace", "Storj (grant)", "S3 (SSO)", "Google Cloud (service account)", "Cloudflare R2"}

// Convert makes an export of the site manager: new IDs, no host key (Vigie asks for each one at
// its first connection), no password since none was read.
func Convert(top sitemanager.Folder, at time.Time, app string) (export.File, []Skipped) {
	c := converter{file: export.File{
		Exported: at.UTC().Truncate(time.Second), App: app,
		Folders: []export.Folder{}, Hosts: []export.Host{}, Sites: []export.Site{}, Ignores: []export.Ignore{},
	}}
	c.folder(top, "", "")
	return c.file, c.skipped
}

type converter struct {
	file    export.File
	skipped []Skipped
}

// folder adds the content of a folder: its folders, then its servers, in their order.
func (c *converter) folder(f sitemanager.Folder, id, path string) {
	for i, sub := range f.Folders {
		name := nodeName(sub.Name)
		subID := uuid.NewV7().String()
		c.file.Folders = append(c.file.Folders, export.Folder{ID: subID, Parent: id, Name: name, Color: "none", Position: i})
		c.folder(sub, subID, join(path, name))
	}
	position := 0
	for _, s := range f.Servers {
		name := nodeName(s.Name)
		if name == "" {
			name = nodeName(s.Host)
		}
		skip := func(r Reason, protocol string) {
			c.skipped = append(c.skipped, Skipped{Path: join(path, name), Reason: r, Protocol: protocol})
		}
		switch {
		case s.Protocol != sitemanager.ProtocolSFTP:
			skip(ReasonProtocol, protocolName(s.Protocol))
			continue
		case s.Host == "":
			skip(ReasonNoHost, "")
			continue
		case s.User == "" || s.Logontype == sitemanager.LogonAnonymous:
			skip(ReasonNoUser, "")
			continue
		}
		h := export.Host{
			ID: uuid.NewV7().String(), Folder: id, Name: name, Hostname: s.Host, User: s.User,
			TrustedKeys: []export.HostKey{}, Color: colorName(s.Colour), Notes: s.Comments, Position: position,
		}
		if s.Port != 22 && s.Port > 0 && s.Port <= 65535 {
			h.Port = s.Port
		}
		if s.Logontype == sitemanager.LogonKey {
			h.KeyPath = s.Keyfile
		}
		c.file.Hosts = append(c.file.Hosts, h)
		position++
	}
}

// nodeName is a name Vigie takes: a slash would split a path in Vigie, so it becomes a dash.
func nodeName(name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, "/", "-"))
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "/" + name
}

func colorName(i int) string {
	if i < 0 || i >= len(colors) {
		return "none"
	}
	return colors[i]
}

func protocolName(i int) string {
	if i < 0 || i >= len(protocols) {
		return "unknown"
	}
	return protocols[i]
}
