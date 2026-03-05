# Changelog

All notable changes to logcurse will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.5.0] - 2026-03-05

### Added

- Read-write mode for the web viewer: `logcurse --serve --rw <file|dir>` enables creating, editing, and deleting comments directly in the browser
- "Add Comment" button appears when lines are selected in read-write mode, with an inline form for writing comment body
- Inline edit and delete controls on each comment block (visible only in read-write mode)
- Author name prompt on first comment creation, stored in browser `localStorage` and used for comment IDs
- Author-based comment IDs (`chip1`, `chip2`, ...) when an author is set; timestamp-based IDs (`w1741193400`) when anonymous
- "READ-WRITE" badge in web viewer header when `--rw` is active
- `/api/config` endpoint returning server configuration (rw mode, version)
- `/api/comments/create`, `/api/comments/update`, `/api/comments/delete` endpoints (only registered when `--rw` is enabled; return 404 otherwise)
- New comments are inserted in sorted order by starting line number in the YAML sidecar file

## [0.4.0] - 2026-03-05

### Added

- GitHub-style line number selection in the web viewer: click a line number to select it, shift+click to select a range
- URL hash updates (`#L10` or `#L10-L25`) for shareable links that highlight and scroll to selected lines
- Opening a URL with a line hash loads the lines, highlights them, and scrolls into view

### Fixed

- Comment hover highlighting now works correctly for lines that haven't been loaded yet (lines load before highlight is applied)

## [0.3.0] - 2026-03-04

### Added

- Added ability to download comment file

## [0.2.2] - 2026-03-04

### Added

- Version string links to project webpage

### Fixed

- Fixed single line scroll on smaller browser windows

## [0.2.1] - 2026-03-04

No change, GitHub pages required a new release.

## [0.2.0] - 2026-03-04

### Added

- Directory serving mode: `logcurse --serve ./logs/` serves a browsable file listing with annotation status, comment counts, and file sizes
- Click any file in the directory listing to open it in the web viewer
- "Back to directory" navigation link when viewing a file in directory mode
- Download button in the web viewer header to save the source file
- `/api/files` endpoint returning JSON list of files with annotation metadata
- `/api/download` endpoint for file downloads in both single-file and directory modes
- Path traversal protection for all directory-mode endpoints

### Changed

- Web viewer shows relative filename instead of full absolute path when serving a directory
- Updated docs site Docker section with two-column layout showing `docker run` and `docker-compose.yml` examples
- Updated README, docs site, and llms.txt with directory serving usage and examples

## [0.1.0] - 2026-03-04

### Added

- Windows installer (Inno Setup) published to GitHub Releases — installs to Program Files, adds to PATH, and registers in Add/Remove Programs
- Docker image published to GHCR (`ghcr.io/wasson-ece/logcurse`) for serving the web viewer without installing Go
- Single-line comment syntax (`logcurse -n 42 file.log`)
- `--version` flag
- Version display in web viewer header
- Gap separators in web viewer for non-contiguous loaded ranges with load up/down buttons
- SHA256 checksums in release artifacts
- CI workflow running tests and vet on pushes and PRs
- LICENSE file (MIT)
- CONTRIBUTING.md
- This changelog

### Fixed

- Web viewer line ordering bug when scrolling back up through loaded chunks
