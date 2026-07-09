# S.L.O.P. — Simple Lightweight Organizer of Posts

A single-binary Go + SQLite blog with a built-in admin UI. Posts, static pages, theming, and backups — no separate database server.

## Quick start

### Install from a release

Pre-built binaries are on [GitHub Releases](https://github.com/birdhalfbaked/slop/releases). Pick the archive for your platform:

| Platform | Archive |
| --- | --- |
| Linux (x86_64) | `slop-linux-amd64.tar.gz` |
| Linux (ARM64) | `slop-linux-arm64.tar.gz` |
| macOS (Apple Silicon) | `slop-darwin-arm64.tar.gz` |
| macOS (Intel) | `slop-darwin-amd64.tar.gz` |
| Windows | `slop-windows-amd64.zip` |

**Linux (x86_64)**

```bash
curl -fsSL https://github.com/birdhalfbaked/slop/releases/latest/download/slop-linux-amd64.tar.gz | tar -xz
sudo install slop-linux-amd64 /usr/local/bin/slop
slop serve
```

**Linux (ARM64)** — use `slop-linux-arm64.tar.gz` and `slop-linux-arm64` instead.

**macOS (Apple Silicon)**

```bash
curl -fsSL https://github.com/birdhalfbaked/slop/releases/latest/download/slop-darwin-arm64.tar.gz | tar -xz
sudo install slop-darwin-arm64 /usr/local/bin/slop
slop serve
```

**macOS (Intel)** — use `slop-darwin-amd64.tar.gz` and `slop-darwin-amd64` instead.

**Windows (PowerShell)**

```powershell
Invoke-WebRequest -Uri "https://github.com/birdhalfbaked/slop/releases/latest/download/slop-windows-amd64.zip" -OutFile slop.zip
Expand-Archive slop.zip -DestinationPath . -Force
.\slop-windows-amd64.exe serve
```

To install on PATH, move `slop-windows-amd64.exe` to a folder on your PATH and rename it to `slop.exe`.

**Pin a specific version** — replace `latest` with a tag, e.g. `download/v0.1.0/slop-linux-amd64.tar.gz`.

**GitHub CLI**

```bash
gh release download -R birdhalfbaked/slop --pattern 'slop-linux-amd64.tar.gz'
tar -xzf slop-linux-amd64.tar.gz
sudo install slop-linux-amd64 /usr/local/bin/slop
```

### Build from source

Requires **Go 1.22+**. No CGO.

```bash
git clone https://github.com/birdhalfbaked/slop.git && cd slop
go build -o slop ./cmd/slop    # Windows: -o slop.exe
./slop serve
```

### Run

```bash
slop serve
```

Opens **http://localhost:8080** (admin at `/admin`). On first start, log in with `admin` / `admin` — you will be prompted to set your own username and password immediately.

## Where data lives

| OS | Path |
| --- | --- |
| Windows | `%USERPROFILE%\AppData\LocalLow\slop\` |
| macOS / Linux | `~/.slop/` |

Database: `{datadir}/db/blog.db` · Uploads: `{datadir}/static/uploads/` · Fonts: `{datadir}/static/fonts/`

Override with `-db` and `-static` flags.

## Documentation

| Guide | Topics |
| --- | --- |
| [Backups & restore](docs/backups.md) | Export, restore, WordPress import, CLI flags |
| [Theming](docs/theming.md) | Editing templates, CSS/JS, template variables |
| [Deploying](docs/deploying.md) | Production setup, reverse proxy, HTTPS, running as a service |

Run `slop <command> -h` for CLI help (`serve`, `passwd`, `backup`, `import`).

## Development

Go 1.22+ required. `go test ./...` · tag `v*` pushes trigger [release builds](https://github.com/birdhalfbaked/slop/releases). See repo layout in source.

## Security

Admin content is trusted raw HTML/JS. Use HTTPS in production and do not share admin access. See [Deploying](docs/deploying.md).
