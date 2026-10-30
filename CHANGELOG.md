# Changelog

All notable changes to vigie-filezilla are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Convert FileZilla's site manager into a Vigie export: folders, and SFTP servers as hosts with
  their name, host name, port, user, key file, comments and color; the other servers listed as
  skipped. Passwords are never read, nothing is connected to.
- FileZilla's own `sitemanager.xml` by default, an export readable by its owner only with `-o`.
- Vigie's add-on contract: `--vigie-info` (its description, image and setting) and
  `--vigie-export`, with the format of Vigie's module `github.com/MarJC5/vigie/addon` v0.1.0 and its
  conformance checks in the tests; `--file` names the site manager to read.
