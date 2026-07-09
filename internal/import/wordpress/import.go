// Package wordpress imports posts from a WordPress WXR export into SLOP.
package wordpress

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"slop/internal/import/html2md"
	"slop/internal/import/htmlclean"
	"slop/internal/import/wxr"
	"slop/internal/models"
	"slop/internal/render"
)

const maxImageBytes = 20 << 20 // 20 MiB

// Options configures a WordPress import run.
type Options struct {
	IncludeDrafts bool
	DryRun        bool
	SkipImages    bool
	OnConflict    string // skip, suffix, overwrite
}

// Result summarizes an import run.
type Result struct {
	PostsFound    int
	Imported      int
	Skipped       int
	SkippedDrafts int
	SkippedSlug   int
	ImageURLs     int
	Errors        []string
}

// Import reads a WXR file and creates SLOP posts.
func Import(db *sql.DB, uploadDir, wxrPath string, opts Options) (*Result, error) {
	export, err := wxr.ParseFile(wxrPath)
	if err != nil {
		return nil, err
	}

	res := &Result{PostsFound: len(export.Posts)}
	fetcher := newImageFetcher(db, uploadDir, opts.DryRun, opts.SkipImages)

	if opts.OnConflict == "" {
		opts.OnConflict = "suffix"
	}

	for _, wp := range export.Posts {
		if wp.Status == "draft" && !opts.IncludeDrafts {
			res.Skipped++
			res.SkippedDrafts++
			continue
		}
		if wp.Status != "publish" && wp.Status != "draft" {
			res.Skipped++
			continue
		}

		cleaned, err := htmlclean.Clean(wp.Content)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%q: clean: %v", wp.Title, err))
			res.Skipped++
			continue
		}

		urls := extractImageURLs(cleaned)
		for _, u := range urls {
			if _, err := fetcher.fetch(u); err != nil && !opts.SkipImages {
				res.Errors = append(res.Errors, fmt.Sprintf("%q: image %s: %v", wp.Title, u, err))
			}
		}

		body, err := html2md.Convert(cleaned, fetcher.rewrite)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%q: convert: %v", wp.Title, err))
			res.Skipped++
			continue
		}

		rendered, err := render.RenderBody("markdown", body)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%q: render: %v", wp.Title, err))
			res.Skipped++
			continue
		}

		slug := wp.Slug
		if slug == "" {
			slug = slugify(wp.Title)
		}

		var featured string
		if wp.ThumbnailID > 0 {
			if u, ok := export.Attachments[wp.ThumbnailID]; ok {
				if path, err := fetcher.fetch(u); err == nil {
					featured = path
				} else if !opts.SkipImages {
					res.Errors = append(res.Errors, fmt.Sprintf("%q: featured image: %v", wp.Title, err))
				}
			}
		}

		status := "draft"
		if wp.Status == "publish" {
			status = "published"
		}

		desc := wp.Excerpt
		pubAt := wp.PublishedAt
		if pubAt.IsZero() {
			pubAt = time.Now().UTC()
		}
		var publishedAt *time.Time
		if status == "published" {
			t := pubAt
			publishedAt = &t
		}

		p := &models.Post{
			Slug:         slug,
			Title:        wp.Title,
			Image:        featured,
			BodySource:   body,
			BodyFormat:   "markdown",
			RenderedHTML: rendered,
			Description:  desc,
			Status:       status,
			PublishedAt:  publishedAt,
			Tags:         wp.Tags,
		}

		if opts.DryRun {
			res.Imported++
			continue
		}

		slug, action, err := resolveSlug(db, slug, opts.OnConflict)
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("%q: slug: %v", wp.Title, err))
			res.Skipped++
			if action == "skip" {
				res.SkippedSlug++
			}
			continue
		}
		p.Slug = slug

		switch action {
		case "overwrite":
			existing, err := models.GetPostBySlug(db, slug)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%q: lookup: %v", wp.Title, err))
				res.Skipped++
				continue
			}
			if existing != nil {
				p.ID = existing.ID
				if err := models.UpdatePost(db, p); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("%q: update: %v", wp.Title, err))
					res.Skipped++
					continue
				}
				res.Imported++
				continue
			}
			fallthrough
		default:
			if _, err := models.CreatePost(db, p); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("%q: create: %v", wp.Title, err))
				res.Skipped++
				continue
			}
			res.Imported++
		}
	}

	// Deduplicate image URL count (was per-post); recount unique from fetcher
	if fetcher != nil {
		res.ImageURLs = len(fetcher.urlMap)
	}

	return res, nil
}

func resolveSlug(db *sql.DB, slug, onConflict string) (string, string, error) {
	exists, err := models.SlugExists(db, slug, 0)
	if err != nil {
		return "", "", err
	}
	if !exists {
		return slug, "create", nil
	}
	switch onConflict {
	case "skip":
		return slug, "skip", fmt.Errorf("slug %q already exists", slug)
	case "overwrite":
		return slug, "overwrite", nil
	default: // suffix
		for i := 2; ; i++ {
			candidate := fmt.Sprintf("%s-%d", slug, i)
			ok, err := models.SlugExists(db, candidate, 0)
			if err != nil {
				return "", "", err
			}
			if !ok {
				return candidate, "create", nil
			}
		}
	}
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

var imgSrcRe = regexp.MustCompile(`(?i)src\s*=\s*["']([^"']+)["']`)

func extractImageURLs(html string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range imgSrcRe.FindAllStringSubmatch(html, -1) {
		if len(m) < 2 {
			continue
		}
		u := strings.TrimSpace(m[1])
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}

type imageFetcher struct {
	db        *sql.DB
	uploadDir string
	client    *http.Client
	urlMap    map[string]string
	dryRun    bool
	skip      bool
}

func newImageFetcher(db *sql.DB, uploadDir string, dryRun, skip bool) *imageFetcher {
	return &imageFetcher{
		db:        db,
		uploadDir: uploadDir,
		client:    &http.Client{Timeout: 30 * time.Second},
		urlMap:    map[string]string{},
		dryRun:    dryRun,
		skip:      skip,
	}
}

func (f *imageFetcher) rewrite(remote string) string {
	if local, ok := f.urlMap[remote]; ok {
		return local
	}
	return remote
}

func (f *imageFetcher) fetch(remote string) (string, error) {
	if remote == "" {
		return "", fmt.Errorf("empty url")
	}
	if local, ok := f.urlMap[remote]; ok {
		return local, nil
	}
	if strings.HasPrefix(remote, "/static/") {
		f.urlMap[remote] = remote
		return remote, nil
	}
	if f.skip {
		f.urlMap[remote] = remote
		return remote, nil
	}
	if f.dryRun {
		f.urlMap[remote] = remote
		return remote, nil
	}

	if err := os.MkdirAll(f.uploadDir, 0o755); err != nil {
		return "", err
	}

	resp, err := f.client.Get(remote)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	name := filenameFromURL(remote)
	dstPath := filepath.Join(f.uploadDir, name)
	if _, err := os.Stat(dstPath); err == nil {
		name = dedupeFilename(name)
		dstPath = filepath.Join(f.uploadDir, name)
	}

	lim := io.LimitReader(resp.Body, maxImageBytes+1)
	data, err := io.ReadAll(lim)
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxImageBytes {
		return "", fmt.Errorf("image exceeds %d bytes", maxImageBytes)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return "", err
	}

	webPath := "/static/uploads/" + name
	if _, err := models.CreateImage(f.db, name, webPath); err != nil {
		return "", err
	}
	f.urlMap[remote] = webPath
	return webPath, nil
}

var unsafeFileRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func filenameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "image"
	}
	base := filepath.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return "image"
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	stem = unsafeFileRe.ReplaceAllString(stem, "-")
	stem = strings.Trim(stem, "-.")
	if stem == "" {
		stem = "image"
	}
	if ext == "" {
		ext = ".bin"
	}
	return stem + ext
}

func dedupeFilename(name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s-%d%s", stem, time.Now().UnixNano()%100000, ext)
}
