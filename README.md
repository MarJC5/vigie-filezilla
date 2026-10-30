# vigie-filezilla

An add-on of [Vigie](https://github.com/MarJC5/vigie): it converts FileZilla's site manager into a
Vigie export, which Vigie then imports with its preview.

- **Folders** stay folders, with their nesting.
- **SFTP servers** become hosts: name, host name, port, user, key file (its path, for a key logon),
  comments as notes, and color.
- **Other servers** (FTP, FTPS, S3, WebDAV…) and SFTP servers without a user are listed as skipped:
  Vigie connects over SSH as a named user.
- **No sites**: Vigie finds the WordPress installs of a host itself (Find new sites).
- **Never a password**: the `Pass` element of FileZilla's file is stepped over and kept nowhere; the
  tests check that no password of a file reaches the export. No host key either: Vigie asks for each
  one at the host's first connection.
- **Nothing is connected to**: the add-on reads one file and writes another.

## Use

```sh
vigie-filezilla -o filezilla.json
```

Without a path, it reads FileZilla's own `sitemanager.xml`: in the first settings folder that exists
of `$XDG_CONFIG_HOME/filezilla`, `~/.config/filezilla` and `~/.filezilla`, as FileZilla looks for
it on macOS and Linux. Give another path as the last argument. Without `-o`, the export goes to the
standard output; with it, the file is readable by you only. What was left out is said on the
standard error.

Then, in Vigie: **Settings > Data > A Vigie file**, or `vigie import filezilla.json`. The preview
shows what is added and what is already there; nothing is added before you confirm.

The export names your servers and users: keep it somewhere safe, or delete it once imported.

## In Vigie

The add-on keeps Vigie's add-on contract ([docs/addons.md](https://github.com/MarJC5/vigie/blob/main/docs/addons.md)
in Vigie): `--vigie-info` prints its description (title, texts in English and French, image, the
site manager to read as its one setting), and `--vigie-export <path>` writes the export where Vigie
asks. Its format comes from Vigie's module `github.com/MarJC5/vigie/addon`, and its tests run
Vigie's conformance checks (`addontest`).

## Build

Go 1.27 or later. While Vigie's repository is private, Go fetches its module directly:
`GOPRIVATE=github.com/MarJC5/*`, with git access to GitHub.

```sh
make build   # ./vigie-filezilla
make check   # format, vet, tests, vulnerabilities
```

## Format

The add-on reads FileZilla's `sitemanager.xml` as FileZilla 3.71.0 reads it
(`src/commonui/site_manager.cpp`, `src/commonui/xml_file.cpp`), and writes Vigie's export format,
version 1, with the types of Vigie's module: there is no copy of the format here.

## License

MIT. Not affiliated with the FileZilla project. FileZilla's name and logo belong to the FileZilla project; the logo on
Vigie's Add-ons page is the one of [Simple Icons](https://simpleicons.org), used to say which
software the add-on reads.
