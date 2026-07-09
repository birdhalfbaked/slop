# S.L.O.P. — Simple Lightweight Organizer of Posts

A single-binary Go + SQLite blog with a built-in admin UI. Posts, static pages, theming, and backups — no separate database server.

## Quick start

**Install** — download a release for your OS from [GitHub Releases](https://github.com/birdhalfbaked/slop/releases), or build from source:

```bash
git clone https://github.com/birdhalfbaked/slop.git && cd slop
go build -o slop ./cmd/slop    # Windows: -o slop.exe
```

**Run**

```bash
slop serve
```

Opens **http://localhost:8080** (admin at `/admin`). On first start, log in with `admin` / `admin` — you will be prompted to set your own username and password immediately.

**Windows:** `.\slop-windows-amd64.exe serve` after extracting the release zip.

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
