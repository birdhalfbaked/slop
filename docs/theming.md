# Theming

SLOP stores your theme in the database, not as files on disk. On first run, defaults are copied from the embedded theme; after that you edit everything in **Admin → Theme**.

## Editing fragments

Each fragment is a separate template or asset:

| Fragment | Purpose |
| --- | --- |
| Base Layout | HTML shell, `{{.Content}}` wrapper |
| Site Header / Footer | Shared chrome |
| Post Header | Title, date, tags, featured image on posts |
| Home / Post List | Blog index |
| Tag Archive | Posts filtered by tag |
| Single Post | Post body layout |
| Static Page | Standalone pages (About, etc.) |
| 404 Not Found | Missing URL page |
| Theme CSS | Site-wide styles |
| Theme JS | Site-wide scripts |

Edit a fragment and click **Save**. Use **Reset to default** on a fragment to restore the shipped version without affecting others.

## Template data

Templates are Go `html/template` with the variables below.

**Site**

- `.Site.Title`, `.Site.Description`, `.Site.BaseURL`
- `.Site.OGSiteName`, `.Site.TwitterHandle`

**Single post** (post and preview pages)

- `.Post.Date`, `.Post.Title`, `.Post.TitlePlain`, `.Post.Slug`
- `.Post.Image`, `.Post.Tags`, `.Post.Content`

**Static page**

- `.StaticPage.Title`, `.StaticPage.TitlePlain`, `.StaticPage.Slug`, `.StaticPage.Content`

**Post lists** (home, tag archive) — `.Posts` is a slice; each item has the same fields as `.Post`.

**Pagination & tags** (home and tag pages)

- `.Page`, `.TotalPages`, `.HasPrev`, `.HasNext`, `.PrevPage`, `.NextPage`
- `.Tag`, `.TagSlug` (tag archive only)

**Injected head/body blocks**

- `.Meta` — SEO and Open Graph tags
- `.Fonts` — font links and `@font-face` rules
- `.Styles`, `.Scripts` — theme CSS and JS
- `.Content` — rendered inner page (used inside base layout)

## Template helpers

| Function | Use |
| --- | --- |
| `formatDate` | Format a `time.Time`, e.g. `{{formatDate .Post.Date "Jan 2, 2006"}}` |
| `safeHTML` | Render trusted HTML, e.g. `{{safeHTML .Post.Title}}` |
| `join` | Join strings: `{{join ", " .Post.Tags}}` |
| `tagSlug` | URL slug for a tag name |

Post titles may contain HTML — use `{{safeHTML .Post.Title}}` in headings and `.TitlePlain` where you need plain text (e.g. `alt` attributes). Meta/OG titles are always stripped to plain text automatically.

## CSS tips

Default content is wrapped in `.container` (max-width ~720px). To break out to full viewport width from inside post/page content, add a utility class in **Theme CSS**:

```css
.full-bleed {
  width: 100vw;
  margin-left: calc(50% - 50vw);
  margin-right: calc(50% - 50vw);
}
html { overflow-x: clip; }
```

Then use `<div class="full-bleed">...</div>` in markdown/HTML content.

## SEO settings

**Admin → SEO & Social** sets site-wide defaults (title, description, base URL, posts per page). Per-post SEO fields are in the post editor. Set **Base URL** to your public `https://` URL when deploying — see [Deploying](deploying.md).

## Publishing & URLs

- Posts and published pages: `/{slug}`
- Tag archives: `/tag/{slug}` with optional `?page=N`
- Home pagination: `/?page=N` (page size in SEO settings, default 10)
- Scheduled posts: set **Publish date** in the editor; future dates stay hidden until then
