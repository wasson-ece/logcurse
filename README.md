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
# serve a single file
logcurse --serve server.log

# serve an entire directory of log files
logcurse --serve ./logs/

# use a custom port
logcurse --serve --port 9090 server.log

# enable read-write mode (add/edit/delete comments in the browser)
logcurse --serve --rw server.log
```

Serves a web viewer at `http://localhost:8080` (default). Lines load on demand in chunks, so this works with large files. Click a line number to select it (shift+click for a range) — the URL updates to `#L10` or `#L10-L25` so you can share links that highlight and scroll to specific lines.

When given a directory, logcurse shows a file listing page with annotation status. Annotated files (those with `.yml` sidecars) are sorted to the top with comment counts. Click any file to open it in the viewer.

#### Read-write mode

The `--rw` flag enables creating, editing, and deleting comments directly in the browser. Without it, the web viewer is read-only and no write endpoints are registered.

In read-write mode:
- Select lines and click "Add Comment" to create a new comment
- Use the [EDIT] and [DEL] buttons on each comment to modify or remove it
- On your first comment, you'll be prompted for a name which is stored in your browser and used for comment IDs (e.g. `chip1`, `chip2`)

## Install

### Docker

```bash
# serve a single file
docker run -p 8080:8080 -v /path/to/logs:/data ghcr.io/wasson-ece/logcurse /data/server.log

# serve a directory
docker run -p 8080:8080 -v /path/to/logs:/data ghcr.io/wasson-ece/logcurse /data/
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

## Agent Usage

logcurse publishes machine-readable documentation for LLMs and AI coding agents at [logcurse.wasson-ece.dev/llms.txt](https://logcurse.wasson-ece.dev/llms.txt).

When instructing an LLM to use logcurse, include something like this in your prompt:

```
logcurse is a CLI tool for annotating log files with line-range comments
stored in YAML sidecar files. For full usage and API documentation, read
https://logcurse.wasson-ece.dev/llms.txt
```

## License

MIT
