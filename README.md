# logcurse

Annotate log files and text files with line-range comments. Comments are stored in YAML sidecar files alongside the original — the source file is never modified.

## Usage

### Add a comment

```bash
# comment on a range of lines
logcurse -n '140,160p' server.log

# comment on a single line
logcurse -n 42 server.log
```

Opens your `$EDITOR` with the referenced lines shown as context. Write your comment, save, and close. The comment is stored in `server.log.yml`.

The range uses sed-style syntax: `'start,endp'` where lines are 1-indexed and inclusive. A bare number like `42` or `42p` targets a single line.

### View in the terminal

```bash
logcurse server.log
```

Opens a dual-pane TUI. The left pane shows the file with line numbers; the right pane shows comments aligned to their line ranges.

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between panes |
| `n` / `p` | Jump to next / previous comment |
| `↑` `↓` `j` `k` | Scroll |
| `s` | Toggle vertical / horizontal split |
| `?` | Help |
| `q` | Quit |

### View in the browser

```bash
logcurse --serve server.log
logcurse --serve --port 9090 server.log
```

Serves a web viewer at `http://localhost:8080` (default). Lines load on demand in chunks, so this works with large files.

## Install

### Docker

```bash
docker run -p 8080:8080 -v /path/to/logs:/data ghcr.io/wasson-ece/logcurse /data/server.log
```

### Windows Installer

Download `logcurse-setup-amd64.exe` from [GitHub Releases](https://github.com/wasson-ece/logcurse/releases). The installer adds logcurse to your PATH and registers an uninstaller in Add/Remove Programs.

### Go install

Requires Go 1.24+.

```bash
go install github.com/wasson-ece/logcurse@latest
```

### Build from source

```bash
git clone https://github.com/wasson-ece/logcurse.git
cd logcurse
go build -v
```

## How it works

Comments live in a YAML sidecar file (`<file>.yml`) next to the original:

```yaml
version: 1
source_file: server.log
comments:
  - id: "c1"
    range:
      start: 140
      end: 160
    content_hash: "sha256:a3f2b8..."
    author: ""
    created: "2026-03-04T10:30:00Z"
    updated: "2026-03-04T10:30:00Z"
    body: |
      Connection timeout during deploy.
      Root cause was the DB migration running long.
```

Each comment records a SHA-256 hash of the lines it references. If the source file changes, logcurse detects the drift and flags affected comments — it never silently modifies the YAML.

## Editor selection

logcurse checks these in order:

1. `$VISUAL`
2. `$EDITOR`
3. `nano` (macOS/Linux) or `notepad` (Windows)

## License

MIT
