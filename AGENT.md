# AGENT.md — logcurse

## What is this?

A Go CLI tool for annotating log/text files with line-range comments stored in YAML sidecar files. Three modes: add comments via `$EDITOR`, view in a TUI, or serve a web viewer.

## Quick Reference

```bash
go build -v                          # build
go test ./...                        # run all tests (12 tests across 3 packages)
go vet ./...                         # lint

logcurse file.txt                    # TUI viewer (default)
logcurse -n '140,160p' file.txt      # add comment on lines 140-160
logcurse --serve file.txt            # web viewer on :8080
logcurse --serve --port 9090 file.txt
```

## Project Layout

```
main.go                              # Flag parsing, dispatches to editor/tui/web
internal/
  model/comment.go                   # CommentFile/Comment structs, YAML load/save, drift check
  fileutil/fileutil.go               # ReadLines, ContentHash, CountLines, LineIndex
  editor/editor.go                   # Parse sed-style range, launch $EDITOR, write YAML sidecar
  viewer/tui/tui.go                  # Bubble Tea dual-pane viewer (file | comments)
  viewer/tui/keymap.go               # Key bindings
  viewer/web/server.go               # HTTP server, go:embed static assets
  viewer/web/handlers.go             # /api/lines, /api/comments JSON endpoints
  viewer/web/static/                 # index.html, app.js, style.css
```

## Architecture

- **`fileutil`** — Low-level file I/O. Handles line reading, SHA-256 hashing, and `LineIndex` (byte-offset index for O(1) seeking into large files). No dependencies on other internal packages.
- **`model`** — Data layer. Defines the YAML sidecar schema, load/save, drift detection. Depends on `fileutil` for hashing.
- **`editor`** — Comment creation workflow. Parses sed-style ranges, builds a temp file with `#`-prefixed context lines, launches `$EDITOR`/`$VISUAL`, extracts the comment body, and appends to the YAML sidecar. Depends on `fileutil` and `model`.
- **`viewer/tui`** — Bubble Tea TUI. Two viewports (file + comments), focus switching, comment navigation, layout toggle. Depends on `fileutil` and `model`.
- **`viewer/web`** — HTTP server. Embeds static assets via `go:embed`. Uses `LineIndex` for chunked line delivery to the JS frontend. Depends on `fileutil` and `model`.

Dependency graph: `fileutil` ← `model` ← `editor`, `viewer/tui`, `viewer/web`

## YAML Sidecar Format

Sidecar lives at `<file>.yml` (e.g., `loga.txt` → `loga.txt.yml`).

```yaml
version: 1
source_file: loga.txt
comments:
  - id: "c1"                        # sequential: c1, c2, c3...
    range:
      start: 140                     # 1-indexed, inclusive
      end: 160
    content_hash: "sha256:a3f2b8..."  # SHA-256 of raw bytes of lines at comment time
    author: ""
    created: "2026-03-04T10:30:00Z"  # RFC 3339
    updated: "2026-03-04T10:30:00Z"
    body: |
      Comment text here
```

**Drift detection**: On load, each comment's hash is recomputed against the current file. Mismatches set a transient `Drifted` bool (not persisted). The YAML is never silently modified.

## Conventions

- **Line numbers are 1-indexed and inclusive** everywhere (ranges, APIs, display).
- **Errors are wrapped** with `fmt.Errorf("context: %w", err)` and propagated up. `main.go` prints to stderr and exits.
- **Scanner buffer** is 1MB (`1024*1024`) to handle long lines.
- **No external dependencies** beyond Charm (bubbletea, bubbles, lipgloss) and `gopkg.in/yaml.v3`.
- Web frontend is vanilla JS — no build step, no framework.
- Static assets are embedded at compile time via `//go:embed`.

## Tests

Tests exist for the three core packages (`fileutil`, `model`, `editor`). The TUI and web packages have no tests (interactive/visual). All tests use `t.TempDir()` for isolation.

Key test cases:
- `fileutil`: line reading, hashing consistency, LineIndex build + binary roundtrip
- `model`: YAML roundtrip, ID generation, drift detection (modify file → verify `Drifted` flag)
- `editor`: sed-style range parsing (valid + invalid), comment extraction from `#`-prefixed temp file

## TUI Key Bindings

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `Tab` | Switch focus between file and comment pane |
| `n` / `p` | Next / previous comment |
| `↑/k` `↓/j` | Scroll |
| `PgUp` `PgDn` `Ctrl+U` `Ctrl+D` | Page scroll |
| `Home/g` `End/G` | Top / bottom |
| `s` | Toggle vertical/horizontal layout |
| `?` | Toggle help bar |

## Web API

| Endpoint | Response |
|----------|----------|
| `GET /api/lines?start=N&end=M` | `{ lines: [{number, text}], total_lines, start, end }` |
| `GET /api/comments` | `{ comments: [{id, range_start, range_end, body, drifted, ...}], source_file }` |

The JS frontend fetches 200-line chunks on scroll, buffering 2 chunks ahead/behind.
