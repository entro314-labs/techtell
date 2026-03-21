# techtell

`techtell` is a terminal UI (TUI) tool that scans a developer workspace and summarizes its **tech DNA**:

- detected projects
- primary stacks/frameworks
- languages
- package managers
- notable tags/libraries

It presents results in an interactive dashboard and can export reports as Markdown or HTML.

## What it does

- Walks a workspace and discovers project roots (for example: `go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, `Dockerfile`, and more).
- Applies signature-based detection to infer:
  - language
  - stack/framework
  - package manager
  - related tags
- Handles monorepos by continuing traversal when workspace markers are detected (for example `pnpm-workspace.yaml`, `go.work`, `lerna.json`).
- Shows:
  - dashboard view with top stats and distributions
  - inventory table of discovered projects
- Exports reports:
  - Markdown (`s`)
  - HTML (`h`)

## Requirements

- Go **1.26+**
- A terminal that supports interactive TUIs

## Quick start

### Install from source

```bash
go install github.com/entro314-labs/techtell/cmd/techtell@latest
```

### Build locally

```bash
git clone https://github.com/entro314-labs/techtell.git
cd techtell
make build
```

### Run

```bash
./techtell
```

Or, if installed globally:

```bash
techtell
```

Check version:

```bash
techtell --version
```

## Using the TUI

1. Launch `techtell`.
2. At the welcome screen, enter a path to scan (or leave empty to scan current directory).
3. Press `Enter` to start analysis.
4. In results:
   - use `Tab` to switch between **Dashboard** and **Projects**
   - use `s` to save Markdown report
   - use `h` to save HTML report

### Key bindings

| Key | Action |
|---|---|
| `Enter` | Start scan / select |
| `Tab` / `Shift+Tab` | Switch result views |
| `s` | Save Markdown report |
| `h` | Save HTML report |
| `Esc` / `Backspace` | Back |
| `q` / `Ctrl+C` | Quit |

## Output files

Reports are saved in the current working directory with timestamped names:

- `techtell_report_<unix>.md`
- `techtell_report_<unix>.html`

## Development

Useful Make targets:

```bash
make help           # list targets
make setup          # install tools + deps
make build          # build binary
make run            # build + print version
make test           # run tests (race + coverage)
make lint           # run golangci-lint (or go vet fallback)
make check          # fmt + lint + test
make build-all      # cross-platform builds
make release-check  # validate goreleaser config
make release-snapshot
```

## Docker

Build and run version check:

```bash
make docker-build
make docker-run
```

Interactive scan example (mount your workspace):

```bash
docker run --rm -it -v "$PWD:/workspace" techtell:latest
```

## CI

The CI pipeline runs:

- tests on Linux/macOS/Windows
- linting (`golangci-lint`)
- security scanning (`govulncheck`)
- build verification
- GoReleaser config and snapshot checks
- Docker build verification

## Security

See [`SECURITY.md`](./SECURITY.md) for vulnerability reporting and release verification guidance.
