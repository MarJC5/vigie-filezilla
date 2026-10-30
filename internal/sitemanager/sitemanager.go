// Package sitemanager reads FileZilla's site manager, sitemanager.xml: its folders and servers,
// never their passwords. The format is FileZilla's own, as its source reads it (3.71.0:
// src/commonui/site_manager.cpp and xml_file.cpp).
package sitemanager

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrNotASiteManager is returned for a file whose root is not FileZilla3.
var ErrNotASiteManager = errors.New("not a FileZilla site manager")

// maxSize bounds what is read: far above a site manager with thousands of servers.
const maxSize = 64 << 20

// The protocols of FileZilla's ServerProtocol (src/include/server.h), by number.
const (
	ProtocolFTP  = 0
	ProtocolSFTP = 1
)

// The logon types of FileZilla's LogonType (src/include/server.h), by number.
const (
	LogonAnonymous = 0
	LogonKey       = 5
)

// Folder is a folder of the site manager; the top one holds what is not in a folder.
type Folder struct {
	Name    string
	Folders []Folder
	Servers []Server
}

// Server is a site of FileZilla: one account on one server. Its password is not in it.
type Server struct {
	Name      string
	Host      string
	Port      int
	Protocol  int
	Logontype int
	User      string
	Keyfile   string // the path of a key on this computer, for a key logon
	Comments  string
	Colour    int // 0 none, then red, green, blue, yellow, cyan, magenta, orange
}

// Read reads a site manager. The Pass element of each server is skipped: the decoder steps over
// its text to find its end, and keeps none of it. So is every element Vigie has no use for.
func Read(r io.Reader) (Folder, error) {
	d := xml.NewDecoder(io.LimitReader(r, maxSize))
	root, err := firstElement(d)
	if err != nil {
		return Folder{}, err
	}
	if root.Name.Local != "FileZilla3" {
		return Folder{}, fmt.Errorf("root %q: %w", root.Name.Local, ErrNotASiteManager)
	}
	var top Folder
	for {
		start, err := nextChild(d)
		if err != nil {
			return Folder{}, err
		}
		if start == nil {
			// FileZilla loads nothing from a file without Servers.
			return top, nil
		}
		if start.Name.Local != "Servers" {
			if err := d.Skip(); err != nil {
				return Folder{}, err
			}
			continue
		}
		if err := readFolder(d, &top); err != nil {
			return Folder{}, err
		}
	}
}

// firstElement returns the root element, after the XML declaration and comments.
func firstElement(d *xml.Decoder) (xml.StartElement, error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return xml.StartElement{}, fmt.Errorf("%w: %w", ErrNotASiteManager, err)
		}
		if start, ok := tok.(xml.StartElement); ok {
			return start, nil
		}
	}
}

// nextChild returns the next child element of the current one, or nil at its end; the text
// between children is dropped.
func nextChild(d *xml.Decoder) (*xml.StartElement, error) {
	for {
		tok, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			return &t, nil
		case xml.EndElement:
			return nil, nil
		}
	}
}

// readFolder reads the content of a Servers or Folder element into f. A folder is named by its own
// text; one without a name is left out, as FileZilla does.
func readFolder(d *xml.Decoder, f *Folder) error {
	var name strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.CharData:
			name.Write(t)
		case xml.EndElement:
			f.Name = strings.TrimSpace(name.String())
			return nil
		case xml.StartElement:
			switch t.Name.Local {
			case "Folder":
				var sub Folder
				if err := readFolder(d, &sub); err != nil {
					return err
				}
				if sub.Name != "" {
					f.Folders = append(f.Folders, sub)
				}
			case "Server":
				s, err := readServer(d)
				if err != nil {
					return err
				}
				f.Servers = append(f.Servers, s)
			default:
				if err := d.Skip(); err != nil {
					return err
				}
			}
		}
	}
}

// readServer reads the elements of a Server that Vigie uses. Its name is the Name element, or
// else its own text, as in files of older FileZilla versions.
func readServer(d *xml.Decoder) (Server, error) {
	var s Server
	var own strings.Builder
	for {
		tok, err := d.Token()
		if err != nil {
			return Server{}, err
		}
		switch t := tok.(type) {
		case xml.CharData:
			own.Write(t)
		case xml.EndElement:
			if s.Name == "" {
				s.Name = strings.TrimSpace(own.String())
			}
			return s, nil
		case xml.StartElement:
			if !wanted[t.Name.Local] {
				// Pass among them: stepped over, nothing of it kept.
				if err := d.Skip(); err != nil {
					return Server{}, err
				}
				continue
			}
			var text string
			if err := d.DecodeElement(&text, &t); err != nil {
				return Server{}, err
			}
			s.set(t.Name.Local, text)
		}
	}
}

// The elements of a Server read; every other one, Pass first, is skipped.
var wanted = map[string]bool{
	"Name": true, "Host": true, "Port": true, "Protocol": true, "Logontype": true,
	"User": true, "Keyfile": true, "Comments": true, "Colour": true,
}

func (s *Server) set(name, text string) {
	number := func() int {
		n, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			return -1
		}
		return n
	}
	switch name {
	case "Name":
		s.Name = strings.TrimSpace(text)
	case "Host":
		s.Host = strings.TrimSpace(text)
	case "Port":
		s.Port = number()
	case "Protocol":
		s.Protocol = number()
	case "Logontype":
		s.Logontype = number()
	case "User":
		s.User = text
	case "Keyfile":
		s.Keyfile = text
	case "Comments":
		s.Comments = text
	case "Colour":
		s.Colour = number()
	}
}

// DefaultPath is where FileZilla keeps its site manager on macOS and Linux: in the first settings
// folder that exists of $XDG_CONFIG_HOME/filezilla, ~/.config/filezilla and ~/.filezilla
// (src/commonui/fz_paths.cpp); "" when none does.
func DefaultPath(getenv func(string) string) string {
	var dirs []string
	if xdg := getenv("XDG_CONFIG_HOME"); filepath.IsAbs(xdg) {
		dirs = append(dirs, filepath.Join(xdg, "filezilla"))
	}
	if home := getenv("HOME"); filepath.IsAbs(home) {
		dirs = append(dirs, filepath.Join(home, ".config", "filezilla"), filepath.Join(home, ".filezilla"))
	}
	for _, dir := range dirs {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return filepath.Join(dir, "sitemanager.xml")
		}
	}
	return ""
}
