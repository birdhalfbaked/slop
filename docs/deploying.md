# Deploying

SLOP is a single process that speaks HTTP. In production, run it bound to localhost and put a **web server** (such as [nginx](https://nginx.org/)) in front to terminate TLS and proxy requests to SLOP.

This guide describes what to plan for — not step-by-step nginx or Docker configuration.

## Architecture

```text
Internet → web server (HTTPS) → SLOP (HTTP on localhost)
```

1. SLOP listens on a loopback address only.
2. Your web server handles public ports 80/443, certificates, and reverse proxying to SLOP.
3. Visitors never connect to SLOP directly.

## Running SLOP

Bind to localhost so the process is not exposed on the network:

```bash
slop serve -addr 127.0.0.1:8080
```

Choose a port that matches what your reverse proxy expects.

### Running as a service (Linux)

Use systemd (or your platform’s equivalent) so SLOP starts on boot and restarts on failure:

```ini
[Unit]
Description=SLOP blog
After=network.target

[Service]
Type=simple
User=slop
ExecStart=/usr/local/bin/slop serve -addr 127.0.0.1:8080
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Install the binary (e.g. to `/usr/local/bin/slop`), create a dedicated user, and enable the unit with `systemctl enable --now slop`.

On Windows, run `slop serve` as a scheduled task or service using your preferred tooling.

## HTTPS / SSL certificates

Serve the public site over **HTTPS**. Configure TLS on your web server (nginx, Caddy, Apache, etc.).

If you do not already have certificates from your host or CDN:

- **[Let’s Encrypt](https://letsencrypt.org/)** provides free automated certificates. Many nginx/Caddy setups integrate with it via [Certbot](https://certbot.eff.org/) or built-in ACME support.

Renewal and exact setup depend on your OS and web server — follow your server’s documentation.

## After going live

1. **Base URL** — In **Admin → SEO & Social**, set **Base URL** to your public URL (e.g. `https://blog.example.com`) so Open Graph and canonical URLs are correct.

2. **Backups** — Schedule `slop backup --full` and store archives off the server. See [Backups & restore](backups.md).

3. **Admin access** — The admin UI is powerful and serves unsanitized HTML/JS in posts. Restrict who can log in; consider network-level restrictions on `/admin` if your proxy supports it.

4. **Firewall** — Only the web server needs ports 80/443 open. SLOP’s port should not be reachable from the internet.

## Session cookies

SLOP sets an `HttpOnly`, `SameSite=Lax` session cookie. HTTPS in front of the app is required for production so credentials and cookies are not sent in cleartext.

## What SLOP does not include

- No built-in TLS — your web server handles HTTPS.
- No container or orchestration recipes — run the binary directly or wrap it yourself.
- No multi-node scaling — one SQLite database per instance; scale by running a single writer.

For backup and migration details, see [Backups & restore](backups.md).
