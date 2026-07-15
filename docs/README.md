# Documentation

User guides for operating SLOP after installation.

| Guide | Description |
| --- | --- |
| [Backups & restore](backups.md) | Exporting your site, restoring from backup, importing from WordPress |
| [Theming](theming.md) | Customizing layouts, CSS, and JavaScript from the admin |
| [Deploying](deploying.md) | Running SLOP in production behind a web server with HTTPS |

## Benchmarks

To run benchmarks locally:

```bash
go test ./... -run '^$' -bench . -benchmem
```

Back to [README](../README.md).
