# Backups & restore

SLOP can export your entire site (database, theme, posts, pages, settings) and restore it later. Uploaded images and fonts are included in **full** backups.

## Admin UI (easiest)

Open **Admin → Backup**.

**Export**

- **Download database (.db)** — SQLite file only. Theme and content are in the DB; image *paths* are included but files are not.
- **Download full backup (.zip)** — database plus everything in `static/uploads/` and `static/fonts/`. Use this before major changes or when moving servers.

**Restore**

- Upload a `.db` or `.zip` backup created by SLOP.
- Confirm the checkbox — this **replaces** posts, pages, theme, and settings from the backup.
- Your **current admin login is preserved** (credentials from the backup are not applied).
- Full zip restores also replace uploaded images and fonts.

Restore from the admin UI works while the server is running (the database is briefly disconnected during restore).

## Security note

Backups restore **theme templates**, **CSS/JS**, and post/page content. Content is intentionally allowed to contain raw HTML and JavaScript.

Only restore backups you created or otherwise fully trust.

## CLI backup

Default paths use your [data directory](../README.md#where-data-lives) (`~/.slop/...` or Windows `%USERPROFILE%\AppData\LocalLow\slop\`).

```bash
# Database only → backups/slop-<timestamp>.db (relative to current directory)
slop backup

# Database + uploads + fonts → backups/slop-<timestamp>.zip
slop backup --full

# Custom output folder
slop backup --full -out /path/to/backups
```

Backups use SQLite `VACUUM INTO` and are safe to run while `slop serve` is running.

### `backup` flags

| Flag | Default | Description |
| --- | --- | --- |
| `-db` | `~/.slop/db/blog.db` | SQLite database file |
| `-static` | `~/.slop/static` | Static assets directory |
| `-out` | `backups` | Output directory (relative to cwd) |
| `-full` | `false` | Include uploads and fonts in a `.zip` |

## CLI restore

```bash
# Preview without writing
slop import backup --dry-run backups/slop-20260708-120000.zip

# Restore database only
slop import backup --force backups/slop-20260708-120000.db

# Restore database + uploads + fonts
slop import backup --force backups/slop-20260708-120000.zip
```

**Stop `slop serve` before CLI restore** — the database file cannot be replaced while the process holds it open. Use Admin → Backup if you need to restore without stopping the server.

Restore **replaces** posts, pages, theme, and settings. **Current admin users and sessions are kept.**

### `import backup` flags

| Flag | Default | Description |
| --- | --- | --- |
| `-db` | `~/.slop/db/blog.db` | Restore destination |
| `-static` | `~/.slop/static` | Restore destination for uploads/fonts |
| `--force` | `false` | Required when target files already exist |
| `--dry-run` | `false` | Validate and print summary only |

## Scheduled backups

Schedule `slop backup --full` with cron (Linux/macOS) or Task Scheduler (Windows). Use `-out` to write to a fixed directory. Store copies off the server when possible.

## WordPress import

Import posts from a WordPress WXR export (WordPress admin → **Tools → Export**). This adds posts to your existing site — it does **not** replace the database or change admin credentials.

```bash
slop import wordpress --dry-run --skip-images export.xml
slop import wordpress export.xml
```

| Flag | Default | Description |
| --- | --- | --- |
| `-db` | `~/.slop/db/blog.db` | SQLite database file |
| `-static` | `~/.slop/static` | Static assets directory |
| `--drafts` | `false` | Also import draft posts |
| `--dry-run` | `false` | Print summary without writing |
| `-on-conflict` | `suffix` | When slug exists: `skip`, `suffix`, or `overwrite` |
| `--skip-images` | `false` | Do not download images |

Images are downloaded into `static/uploads/`. Content is converted to markdown (with HTML for figures and some inline tags).

A minimal WXR sample for development and tests lives at `internal/import/wxr/testdata/sample.wxr.xml`.

## Password recovery

If you are locked out of admin, reset a password from the CLI:

```bash
slop passwd -user admin
```

Or non-interactively:

```bash
# Linux / macOS
SLOP_ADMIN_PASSWORD=changeme slop passwd -user admin
```

```powershell
# Windows PowerShell
$env:SLOP_ADMIN_PASSWORD = "changeme"
slop passwd -user admin
```

Users set via `passwd` are not forced to change credentials on next login.
