package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"slop/internal/auth"
	"slop/internal/models"
	"slop/internal/render"
	"slop/internal/theme"
)

// ---- Auth ----

func (app *App) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	if u, _ := auth.CurrentUser(r, app.DB); u != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	app.renderLogin(w, "")
}

func (app *App) renderLogin(w http.ResponseWriter, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = app.login.ExecuteTemplate(w, "login.html", map[string]any{"Error": errMsg})
}

func (app *App) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ok, err := auth.Login(w, r, app.DB, r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		app.renderLogin(w, "Invalid username or password.")
		return
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	_ = auth.Logout(w, r, app.DB)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// ---- Account (credential change) ----

func (app *App) renderAccount(w http.ResponseWriter, r *http.Request, errMsg string) {
	user := auth.UserFromContext(r.Context())
	app.renderAdmin(w, r, "account", adminCtx{
		Title:  "Account",
		Active: "account",
		Flash:  r.URL.Query().Get("flash"),
		Data: map[string]any{
			"Username":   user.Username,
			"MustChange": user.MustChange,
			"Error":      errMsg,
		},
	})
}

func (app *App) handleAccount(w http.ResponseWriter, r *http.Request) {
	app.renderAccount(w, r, "")
}

func (app *App) handleAccountSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	user := auth.UserFromContext(r.Context())

	current := r.FormValue("current_password")
	newUsername := strings.TrimSpace(r.FormValue("new_username"))
	newPassword := r.FormValue("new_password")
	confirm := r.FormValue("confirm_password")

	if !auth.CheckPassword(user.PasswordHash, current) {
		app.renderAccount(w, r, "Current password is incorrect.")
		return
	}
	if newUsername == "" {
		app.renderAccount(w, r, "Username cannot be empty.")
		return
	}
	if len(newPassword) < 8 {
		app.renderAccount(w, r, "New password must be at least 8 characters.")
		return
	}
	if newPassword != confirm {
		app.renderAccount(w, r, "New password and confirmation do not match.")
		return
	}
	if user.MustChange && newUsername == "admin" && newPassword == "admin" {
		app.renderAccount(w, r, "Please choose a username and password different from the defaults.")
		return
	}
	taken, err := models.UsernameTaken(app.DB, newUsername, user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if taken {
		app.renderAccount(w, r, "That username is already taken.")
		return
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := models.UpdateCredentials(app.DB, user.ID, newUsername, hash); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin?flash=Credentials+updated", http.StatusSeeOther)
}

// ---- Dashboard ----

func (app *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	all, err := models.ListPosts(app.DB, "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var published, drafts []*models.Post
	for _, p := range all {
		if p.Status == "published" {
			published = append(published, p)
		} else {
			drafts = append(drafts, p)
		}
	}
	app.renderAdmin(w, r, "dashboard", adminCtx{
		Title:  "Dashboard",
		Active: "dashboard",
		Flash:  r.URL.Query().Get("flash"),
		Data: map[string]any{
			"Published": published,
			"Drafts":    drafts,
		},
	})
}

// ---- Posts ----

var adminPostsPerPageChoices = []int{5, 10, 25, 50}

func adminPostsPerPage(r *http.Request) int {
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			for _, c := range adminPostsPerPageChoices {
				if n == c {
					return n
				}
			}
		}
	}
	return 10
}

func (app *App) handlePostsList(w http.ResponseWriter, r *http.Request) {
	perPage := adminPostsPerPage(r)
	page := 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 1 {
		page = p
	}
	total, err := models.CountPosts(app.DB, "")
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * perPage
	posts, err := models.ListPostsPaged(app.DB, "", perPage, offset)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	app.renderAdmin(w, r, "posts", adminCtx{
		Title:  "Posts",
		Active: "posts",
		Flash:  r.URL.Query().Get("flash"),
		Data: map[string]any{
			"Posts":      posts,
			"Page":       page,
			"PerPage":    perPage,
			"TotalPages": totalPages,
			"Total":      total,
			"HasPrev":    page > 1,
			"HasNext":    page < totalPages,
			"PrevPage":   page - 1,
			"NextPage":   page + 1,
		},
	})
}

type editorData struct {
	Post    *models.Post
	Images  []*models.Image
	TagsCSV string
	IsNew   bool
}

func (app *App) handlePostNew(w http.ResponseWriter, r *http.Request) {
	images, _ := models.ListImages(app.DB)
	app.renderAdmin(w, r, "editor", adminCtx{
		Title:  "New Post",
		Active: "posts",
		Data: editorData{
			Post:   &models.Post{BodyFormat: "markdown", Status: "draft"},
			Images: images,
			IsNew:  true,
		},
	})
}

func (app *App) handlePostEdit(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	post, err := models.GetPostByID(app.DB, id)
	if err != nil || post == nil {
		http.NotFound(w, r)
		return
	}
	images, _ := models.ListImages(app.DB)
	app.renderAdmin(w, r, "editor", adminCtx{
		Title:   "Edit Post",
		Active:  "posts",
		Flash:   r.URL.Query().Get("flash"),
		Data: editorData{
			Post:    post,
			Images:  images,
			TagsCSV: strings.Join(post.Tags, ", "),
		},
	})
}

func (app *App) postFromForm(r *http.Request) (*models.Post, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	format := r.FormValue("body_format")
	if format != "html" {
		format = "markdown"
	}
	source := r.FormValue("body_source")
	rendered, err := render.RenderBody(format, source)
	if err != nil {
		return nil, err
	}
	p := &models.Post{
		Title:           strings.TrimSpace(r.FormValue("title")),
		Slug:            slugify(r.FormValue("slug"), r.FormValue("title")),
		Image:           strings.TrimSpace(r.FormValue("image")),
		BodySource:      source,
		BodyFormat:      format,
		RenderedHTML:    rendered,
		Description:     strings.TrimSpace(r.FormValue("description")),
		MetaDescription: strings.TrimSpace(r.FormValue("meta_description")),
		OGTitle:         strings.TrimSpace(r.FormValue("og_title")),
		OGDescription:   strings.TrimSpace(r.FormValue("og_description")),
		OGImage:         strings.TrimSpace(r.FormValue("og_image")),
		Tags:            splitTags(r.FormValue("tags")),
	}
	return p, nil
}

func (app *App) handlePostCreate(w http.ResponseWriter, r *http.Request) {
	p, err := app.postFromForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p.Status = "draft"
	p.Slug = app.uniqueContentSlug(p.Slug, "post", 0, 0)
	pubAt, _ := parsePublishedAt(r.FormValue("published_at"))
	if r.FormValue("action") == "publish" {
		p.Status = "published"
		if pubAt != nil {
			p.PublishedAt = pubAt
		} else {
			now := time.Now().UTC()
			p.PublishedAt = &now
		}
	}
	id, err := models.CreatePost(app.DB, p)
	if err != nil {
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/posts/%d/edit?flash=Saved", id), http.StatusSeeOther)
}

func (app *App) handlePostUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	existing, err := models.GetPostByID(app.DB, id)
	if err != nil || existing == nil {
		http.NotFound(w, r)
		return
	}
	p, err := app.postFromForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p.ID = id
	p.Status = existing.Status
	p.Slug = app.uniqueContentSlug(p.Slug, "post", id, 0)
	pubAt, _ := parsePublishedAt(r.FormValue("published_at"))

	switch r.FormValue("action") {
	case "publish":
		p.Status = "published"
		if pubAt != nil {
			p.PublishedAt = pubAt
		} else if existing.PublishedAt != nil {
			p.PublishedAt = existing.PublishedAt
		} else {
			now := time.Now().UTC()
			p.PublishedAt = &now
		}
	case "unpublish":
		p.Status = "draft"
		p.PublishedAt = existing.PublishedAt
	default:
		if existing.Status == "published" {
			p.Status = "published"
			if pubAt != nil {
				p.PublishedAt = pubAt
			} else {
				p.PublishedAt = existing.PublishedAt
			}
		} else {
			p.PublishedAt = existing.PublishedAt
		}
	}

	if err := models.UpdatePost(app.DB, p); err != nil {
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/posts/%d/edit?flash=Saved", id), http.StatusSeeOther)
}

func (app *App) handlePostDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := models.DeletePost(app.DB, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin?flash=Post+deleted", http.StatusSeeOther)
}

func (app *App) setPostStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	post, err := models.GetPostByID(app.DB, id)
	if err != nil || post == nil {
		http.NotFound(w, r)
		return
	}
	post.Status = status
	if status == "published" && post.PublishedAt == nil {
		now := time.Now().UTC()
		post.PublishedAt = &now
	}
	if err := models.UpdatePost(app.DB, post); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin?flash=Updated", http.StatusSeeOther)
}

func (app *App) handlePostPublish(w http.ResponseWriter, r *http.Request) {
	app.setPostStatus(w, r, "published")
}

func (app *App) handlePostUnpublish(w http.ResponseWriter, r *http.Request) {
	app.setPostStatus(w, r, "draft")
}

func (app *App) handlePreview(w http.ResponseWriter, r *http.Request) {
	p, err := app.postFromForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if p.Slug == "" {
		p.Slug = "preview"
	}
	now := time.Now().UTC()
	p.PublishedAt = &now
	p.Status = "published"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := app.Renderer.RenderPage(w, "post", p, nil, nil, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// ---- Images ----

func (app *App) handleImages(w http.ResponseWriter, r *http.Request) {
	images, err := models.ListImages(app.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	app.renderAdmin(w, r, "images", adminCtx{
		Title:  "Images",
		Active: "images",
		Flash:  r.URL.Query().Get("flash"),
		Data:   map[string]any{"Images": images},
	})
}

// saveUploadedImage saves the uploaded file from the given multipart field to
// the uploads directory and records it in the gallery.
func (app *App) saveUploadedImage(r *http.Request, field string) (*models.Image, error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return nil, err
	}
	file, header, err := r.FormFile(field)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	name := safeFilename(header.Filename)
	if err := os.MkdirAll(app.UploadDir, 0o755); err != nil {
		return nil, err
	}
	dst, err := os.Create(filepath.Join(app.UploadDir, name))
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return nil, err
	}
	webPath := "/static/uploads/" + name
	id, err := models.CreateImage(app.DB, name, webPath)
	if err != nil {
		return nil, err
	}
	return &models.Image{ID: id, Filename: name, Path: webPath}, nil
}

func (app *App) handleImageUpload(w http.ResponseWriter, r *http.Request) {
	if _, err := app.saveUploadedImage(r, "file"); err != nil {
		http.Redirect(w, r, "/admin/images?flash=No+file+selected", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/images?flash=Uploaded", http.StatusSeeOther)
}

// handleImageUploadJSON uploads an image and returns its metadata as JSON, for
// the in-editor image tool.
func (app *App) handleImageUploadJSON(w http.ResponseWriter, r *http.Request) {
	img, err := app.saveUploadedImage(r, "file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "upload failed"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":       img.ID,
		"filename": img.Filename,
		"path":     img.Path,
	})
}

func (app *App) handleImageDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	img, err := models.GetImage(app.DB, id)
	if err != nil || img == nil {
		http.NotFound(w, r)
		return
	}
	_ = os.Remove(filepath.Join(app.UploadDir, img.Filename))
	if err := models.DeleteImage(app.DB, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/images?flash=Deleted", http.StatusSeeOther)
}

// ---- Theme ----

type themeFragmentView struct {
	Key      string
	Label    string
	Language string
	Value    string
}

func (app *App) handleTheme(w http.ResponseWriter, r *http.Request) {
	settings, err := models.GetAllSettings(app.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var frags []themeFragmentView
	for _, f := range theme.Fragments {
		frags = append(frags, themeFragmentView{
			Key:      f.Key,
			Label:    f.Label,
			Language: f.Language,
			Value:    settings[f.Key],
		})
	}
	app.renderAdmin(w, r, "theme", adminCtx{
		Title:  "Theme",
		Active: "theme",
		Flash:  r.URL.Query().Get("flash"),
		Data:   map[string]any{"Fragments": frags, "SiteTitle": settings["site_title"]},
	})
}

func (app *App) handleThemeSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	for _, f := range theme.Fragments {
		if _, ok := r.Form[f.Key]; ok {
			if err := models.SetSetting(app.DB, f.Key, r.FormValue(f.Key)); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	}
	if _, ok := r.Form["site_title"]; ok {
		_ = models.SetSetting(app.DB, "site_title", r.FormValue("site_title"))
	}
	http.Redirect(w, r, "/admin/theme?flash=Theme+saved", http.StatusSeeOther)
}

func (app *App) handleThemeReset(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	key := r.FormValue("key")
	reset := func(k string) error {
		def, ok := app.ThemeDefaults[k]
		if !ok {
			return nil
		}
		return models.SetSetting(app.DB, k, def)
	}
	if key == "all" {
		for _, f := range theme.Fragments {
			if err := reset(f.Key); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	} else if theme.IsFragmentKey(key) {
		if err := reset(key); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
	http.Redirect(w, r, "/admin/theme?flash=Reset+to+default", http.StatusSeeOther)
}

// ---- SEO ----

var seoKeys = []string{"site_title", "site_description", "og_default_image", "og_site_name", "twitter_handle", "base_url", "posts_per_page"}

func (app *App) handleSEO(w http.ResponseWriter, r *http.Request) {
	settings, err := models.GetAllSettings(app.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	values := map[string]string{}
	for _, k := range seoKeys {
		values[k] = settings[k]
	}
	app.renderAdmin(w, r, "seo", adminCtx{
		Title:  "SEO & Social",
		Active: "seo",
		Flash:  r.URL.Query().Get("flash"),
		Data:   map[string]any{"Values": values},
	})
}

func (app *App) handleSEOSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	for _, k := range seoKeys {
		if _, ok := r.Form[k]; ok {
			if err := models.SetSetting(app.DB, k, strings.TrimSpace(r.FormValue(k))); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	}
	http.Redirect(w, r, "/admin/seo?flash=SEO+settings+saved", http.StatusSeeOther)
}

// ---- Fonts ----

func (app *App) handleFonts(w http.ResponseWriter, r *http.Request) {
	fonts, err := models.ListFonts(app.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	app.renderAdmin(w, r, "fonts", adminCtx{
		Title:  "Fonts",
		Active: "fonts",
		Flash:  r.URL.Query().Get("flash"),
		Data:   map[string]any{"Fonts": fonts},
	})
}

func (app *App) handleFontCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	f := &models.Font{
		Name:   strings.TrimSpace(r.FormValue("name")),
		Family: strings.TrimSpace(r.FormValue("family")),
		Weight: strings.TrimSpace(r.FormValue("weight")),
		Style:  strings.TrimSpace(r.FormValue("style")),
		Source: r.FormValue("source"),
	}
	if f.Family == "" {
		f.Family = f.Name
	}

	if f.Source == "file" {
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Redirect(w, r, "/admin/fonts?flash=No+font+file+selected", http.StatusSeeOther)
			return
		}
		defer file.Close()
		name := safeFilename(header.Filename)
		if err := os.MkdirAll(app.FontDir, 0o755); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		dst, err := os.Create(filepath.Join(app.FontDir, name))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		f.Filename = name
	} else {
		f.Source = "cdn"
		f.URL = strings.TrimSpace(r.FormValue("url"))
	}

	if _, err := models.CreateFont(app.DB, f); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/fonts?flash=Font+added", http.StatusSeeOther)
}

func (app *App) handleFontDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	font, err := models.GetFont(app.DB, id)
	if err != nil || font == nil {
		http.NotFound(w, r)
		return
	}
	if font.Source == "file" && font.Filename != "" {
		_ = os.Remove(filepath.Join(app.FontDir, font.Filename))
	}
	if err := models.DeleteFont(app.DB, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/fonts?flash=Font+deleted", http.StatusSeeOther)
}

// ---- Helpers ----

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func splitTags(csv string) []string {
	var out []string
	for _, part := range strings.Split(csv, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(slug, fallback string) string {
	s := strings.TrimSpace(slug)
	if s == "" {
		s = fallback
	}
	s = strings.ToLower(s)
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// uniqueContentSlug ensures slug is unique across posts and pages.
func (app *App) uniqueContentSlug(slug, fallback string, excludePostID, excludePageID int64) string {
	if slug == "" {
		slug = fallback
	}
	candidate := slug
	for i := 2; ; i++ {
		exists, err := models.ContentSlugExists(app.DB, candidate, excludePostID, excludePageID)
		if err != nil || !exists {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", slug, i)
	}
}

var unsafeFileRe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func safeFilename(name string) string {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	stem = unsafeFileRe.ReplaceAllString(stem, "-")
	stem = strings.Trim(stem, "-.")
	if stem == "" {
		stem = "file"
	}
	ext = unsafeFileRe.ReplaceAllString(ext, "")
	return fmt.Sprintf("%d-%s%s", time.Now().UnixNano(), stem, ext)
}

func parsePublishedAt(form string) (*time.Time, error) {
	form = strings.TrimSpace(form)
	if form == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", form, time.Local)
	if err != nil {
		return nil, err
	}
	utc := t.UTC()
	return &utc, nil
}
