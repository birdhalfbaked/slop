package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"slop/internal/models"
	"slop/internal/render"
)

func (app *App) handlePagesList(w http.ResponseWriter, r *http.Request) {
	pages, err := models.ListStaticPages(app.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	total, _ := models.CountStaticPages(app.DB)
	app.renderAdmin(w, r, "pages", adminCtx{
		Title:  "Pages",
		Active: "pages",
		Flash:  r.URL.Query().Get("flash"),
		Data: map[string]any{
			"Pages": pages,
			"Total": total,
		},
	})
}

type pageEditorData struct {
	Page  *models.StaticPage
	IsNew bool
}

func (app *App) handlePageNew(w http.ResponseWriter, r *http.Request) {
	app.renderAdmin(w, r, "page_editor", adminCtx{
		Title:  "New Page",
		Active: "pages",
		Data: pageEditorData{
			Page:  &models.StaticPage{BodyFormat: "markdown", Status: "draft"},
			IsNew: true,
		},
	})
}

func (app *App) handlePageEdit(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	page, err := models.GetStaticPageByID(app.DB, id)
	if err != nil || page == nil {
		http.NotFound(w, r)
		return
	}
	app.renderAdmin(w, r, "page_editor", adminCtx{
		Title:  "Edit Page",
		Active: "pages",
		Flash:  r.URL.Query().Get("flash"),
		Data: pageEditorData{
			Page: page,
		},
	})
}

func (app *App) pageFromForm(r *http.Request) (*models.StaticPage, error) {
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
	return &models.StaticPage{
		Title:        strings.TrimSpace(r.FormValue("title")),
		Slug:         slugify(r.FormValue("slug"), r.FormValue("title")),
		BodySource:   source,
		BodyFormat:   format,
		RenderedHTML: rendered,
	}, nil
}

func (app *App) handlePageCreate(w http.ResponseWriter, r *http.Request) {
	p, err := app.pageFromForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p.Status = "draft"
	p.Slug = app.uniqueContentSlug(p.Slug, "page", 0, 0)
	if r.FormValue("action") == "publish" {
		p.Status = "published"
	}
	id, err := models.CreateStaticPage(app.DB, p)
	if err != nil {
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/pages/%d/edit?flash=Saved", id), http.StatusSeeOther)
}

func (app *App) handlePageUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	existing, err := models.GetStaticPageByID(app.DB, id)
	if err != nil || existing == nil {
		http.NotFound(w, r)
		return
	}
	p, err := app.pageFromForm(r)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p.ID = id
	p.Status = existing.Status
	p.Slug = app.uniqueContentSlug(p.Slug, "page", 0, id)

	switch r.FormValue("action") {
	case "publish":
		p.Status = "published"
	case "unpublish":
		p.Status = "draft"
	}

	if err := models.UpdateStaticPage(app.DB, p); err != nil {
		http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/admin/pages/%d/edit?flash=Saved", id), http.StatusSeeOther)
}

func (app *App) handlePageDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := models.DeleteStaticPage(app.DB, id); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/pages?flash=Page+deleted", http.StatusSeeOther)
}

func (app *App) handlePagePublish(w http.ResponseWriter, r *http.Request) {
	app.setPageStatus(w, r, "published")
}

func (app *App) handlePageUnpublish(w http.ResponseWriter, r *http.Request) {
	app.setPageStatus(w, r, "draft")
}

func (app *App) setPageStatus(w http.ResponseWriter, r *http.Request, status string) {
	id, err := pathID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	page, err := models.GetStaticPageByID(app.DB, id)
	if err != nil || page == nil {
		http.NotFound(w, r)
		return
	}
	page.Status = status
	page.UpdatedAt = time.Now().UTC()
	if err := models.UpdateStaticPage(app.DB, page); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/pages?flash=Updated", http.StatusSeeOther)
}
