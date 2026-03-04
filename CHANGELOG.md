# Changelog

All notable changes to logcurse will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

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
