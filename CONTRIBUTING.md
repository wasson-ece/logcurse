# Contributing to logcurse

Thanks for your interest in contributing! Here's how to get started.

## Development Setup

Requires Go 1.24+.

```bash
git clone https://github.com/wasson-ece/logcurse.git
cd logcurse
go build -v
```

## Running Tests

```bash
go test ./...
go vet ./...
```

Use the `-race` flag to check for data races:

```bash
go test -race ./...
```

## Project Structure

```
main.go                      CLI entry point and flag parsing
internal/
  editor/                    Comment creation workflow ($EDITOR integration)
  fileutil/                  File I/O, line indexing, SHA-256 hashing
  model/                     YAML sidecar data model and drift detection
  viewer/
    tui/                     Terminal UI (Bubble Tea)
    web/                     Web viewer (HTTP server + embedded static assets)
      static/                Frontend JS/CSS/HTML (embedded via go:embed)
help/                        Project website and llms.txt
```

## Making Changes

1. Fork the repo and create a feature branch from `main`.
2. Make your changes. Add or update tests where applicable.
3. Run `go test ./...` and `go vet ./...` before committing.
4. Keep commits focused — one logical change per commit.
5. Open a pull request against `main`.

## Code Style

- Follow standard Go conventions (`gofmt`).
- Error messages should be lowercase and not end with punctuation.
- Wrap errors with context: `fmt.Errorf("reading file: %w", err)`.
- Keep dependencies minimal. Avoid adding new ones unless necessary.

## Reporting Issues

Open an issue on GitHub with:

- What you expected to happen
- What actually happened
- Steps to reproduce
- logcurse version (`logcurse --version`)
