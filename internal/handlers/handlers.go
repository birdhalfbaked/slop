package handlers

import (
	"database/sql"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"slop/internal/auth"
	"slop/internal/models"
	"slop/internal/render"
)

// App holds shared dependencies for HTTP handlers.
type App struct {
	DB            *sql.DB
	DBPath        string
	Renderer      *render.Renderer
	adminPages    map[string]*template.Template
	login         *template.Template
	ThemeDefaults map[string]string
	StaticDir     string
	UploadDir     string
	FontDir       string
	dbMu          sync.Mutex
}

// NewApp constructs the application, parsing embedded admin templates.
func NewApp(db *sql.DB, dbPath string, r *render.Renderer, adminFS fs.FS, themeDefaults map[string]string, staticDir, uploadDir, fontDir string) (*App, error) {
	app := &App{
		DB:            db,
		DBPath:        dbPath,
		Renderer:      r,
		ThemeDefaults: themeDefaults,
		StaticDir:     staticDir,
		UploadDir:     uploadDir,
		FontDir:       fontDir,
		adminPages:    map[string]*template.Template{},
	}

	pages := []string{"dashboard", "posts", "pages", "page_editor", "editor", "images", "theme", "seo", "fonts", "backup", "account"}
	for _, name := range pages {
		t, err := template.New("layout.html").Funcs(adminFuncs).ParseFS(adminFS, "layout.html", name+".html")
		if err != nil {
			return nil, err
		}
		app.adminPages[name] = t
	}
	login, err := template.New("login.html").Funcs(adminFuncs).ParseFS(adminFS, "login.html")
	if err != nil {
		return nil, err
	}
	app.login = login
	return app, nil
}

var adminFuncs = template.FuncMap{
	"join": func(sep string, s []string) string {
		out := ""
		for i, v := range s {
			if i > 0 {
				out += sep
			}
			out += v
		}
		return out
	},
	"date": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format("Jan 2, 2006 15:04")
	},
	"datePtr": func(t *time.Time) string {
		if t == nil || t.IsZero() {
			return ""
		}
		return t.Format("Jan 2, 2006 15:04")
	},
	"datetimeLocal": func(t *time.Time) string {
		if t == nil || t.IsZero() {
			return ""
		}
		return t.Local().Format("2006-01-02T15:04")
	},
}

// Routes builds the HTTP router.
func (app *App) Routes() http.Handler {
	mux := http.NewServeMux()

	// Static assets (uploaded images + fonts).
	fileServer := http.FileServer(http.Dir(app.StaticDir))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Public site.
	mux.HandleFunc("GET /tag/{tag}", app.handleTag)
	mux.HandleFunc("GET /{$}", app.handleHome)
	mux.HandleFunc("GET /{slug}", app.handlePost)

	// Admin: auth.
	mux.HandleFunc("GET /admin/login", app.handleLoginForm)
	mux.HandleFunc("POST /admin/login", app.handleLoginSubmit)
	mux.HandleFunc("POST /admin/logout", app.requireAuth(app.handleLogout))

	// Admin: account / credential change.
	mux.HandleFunc("GET /admin/account", app.requireAuth(app.handleAccount))
	mux.HandleFunc("POST /admin/account", app.requireAuth(app.handleAccountSave))

	// Admin: dashboard.
	mux.HandleFunc("GET /admin", app.requireAuth(app.handleDashboard))
	mux.HandleFunc("GET /admin/{$}", app.requireAuth(app.handleDashboard))

	// Admin: posts.
	mux.HandleFunc("GET /admin/posts", app.requireAuth(app.handlePostsList))
	mux.HandleFunc("GET /admin/posts/{$}", app.requireAuth(app.handlePostsList))
	mux.HandleFunc("GET /admin/posts/new", app.requireAuth(app.handlePostNew))
	mux.HandleFunc("POST /admin/posts", app.requireAuth(app.handlePostCreate))
	mux.HandleFunc("GET /admin/posts/{id}/edit", app.requireAuth(app.handlePostEdit))
	mux.HandleFunc("POST /admin/posts/{id}", app.requireAuth(app.handlePostUpdate))
	mux.HandleFunc("POST /admin/posts/{id}/delete", app.requireAuth(app.handlePostDelete))
	mux.HandleFunc("POST /admin/posts/{id}/publish", app.requireAuth(app.handlePostPublish))
	mux.HandleFunc("POST /admin/posts/{id}/unpublish", app.requireAuth(app.handlePostUnpublish))
	mux.HandleFunc("POST /admin/preview", app.requireAuth(app.handlePreview))

	// Admin: static pages.
	mux.HandleFunc("GET /admin/pages", app.requireAuth(app.handlePagesList))
	mux.HandleFunc("GET /admin/pages/{$}", app.requireAuth(app.handlePagesList))
	mux.HandleFunc("GET /admin/pages/new", app.requireAuth(app.handlePageNew))
	mux.HandleFunc("POST /admin/pages", app.requireAuth(app.handlePageCreate))
	mux.HandleFunc("GET /admin/pages/{id}/edit", app.requireAuth(app.handlePageEdit))
	mux.HandleFunc("POST /admin/pages/{id}", app.requireAuth(app.handlePageUpdate))
	mux.HandleFunc("POST /admin/pages/{id}/delete", app.requireAuth(app.handlePageDelete))
	mux.HandleFunc("POST /admin/pages/{id}/publish", app.requireAuth(app.handlePagePublish))
	mux.HandleFunc("POST /admin/pages/{id}/unpublish", app.requireAuth(app.handlePageUnpublish))

	// Admin: images.
	mux.HandleFunc("GET /admin/images", app.requireAuth(app.handleImages))
	mux.HandleFunc("POST /admin/images", app.requireAuth(app.handleImageUpload))
	mux.HandleFunc("POST /admin/images/upload", app.requireAuth(app.handleImageUploadJSON))
	mux.HandleFunc("POST /admin/images/{id}/delete", app.requireAuth(app.handleImageDelete))

	// Admin: theme.
	mux.HandleFunc("GET /admin/theme", app.requireAuth(app.handleTheme))
	mux.HandleFunc("POST /admin/theme", app.requireAuth(app.handleThemeSave))
	mux.HandleFunc("POST /admin/theme/reset", app.requireAuth(app.handleThemeReset))

	// Admin: SEO.
	mux.HandleFunc("GET /admin/seo", app.requireAuth(app.handleSEO))
	mux.HandleFunc("POST /admin/seo", app.requireAuth(app.handleSEOSave))

	// Admin: fonts.
	mux.HandleFunc("GET /admin/fonts", app.requireAuth(app.handleFonts))
	mux.HandleFunc("POST /admin/fonts", app.requireAuth(app.handleFontCreate))
	mux.HandleFunc("POST /admin/fonts/{id}/delete", app.requireAuth(app.handleFontDelete))

	// Admin: backup & restore.
	mux.HandleFunc("GET /admin/backup", app.requireAuth(app.handleBackupPage))
	mux.HandleFunc("GET /admin/backup/download", app.requireAuth(app.handleBackupDownload))
	mux.HandleFunc("POST /admin/backup/restore", app.requireAuth(app.handleBackupRestore))

	return mux
}

func (app *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		app.dbMu.Lock()
		database := app.DB
		app.dbMu.Unlock()
		if database == nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		user, err := auth.CurrentUser(r, database)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		if user.MustChange && r.URL.Path != "/admin/account" && r.URL.Path != "/admin/logout" {
			http.Redirect(w, r, "/admin/account", http.StatusSeeOther)
			return
		}
		ctx := auth.SetUserContext(r.Context(), user)
		next(w, r.WithContext(ctx))
	}
}

// adminCtx is the root data for admin page templates.
type adminCtx struct {
	Title  string
	Active string
	User   *models.User
	Flash  string
	Error  string
	Data   any
}

func (app *App) renderAdmin(w http.ResponseWriter, r *http.Request, page string, ctx adminCtx) {
	ctx.User = auth.UserFromContext(r.Context())
	t, ok := app.adminPages[page]
	if !ok {
		http.Error(w, "unknown admin page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout.html", ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
