package handlers

import (
	"net/http"
	"strconv"
	"time"

	"slop/internal/models"
	"slop/internal/render"
)

func (app *App) handleHome(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	perPage, err := models.PostsPerPage(app.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	page, totalPages, offset := paginate(r, perPage, func() (int, error) {
		return models.CountPostsPublic(app.DB, now)
	})
	posts, err := models.ListPostsPublicPaged(app.DB, perPage, offset, now)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	opts := &render.PageOpts{
		Page:       page,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	}
	if err := app.Renderer.RenderPage(w, "home", nil, posts, opts, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (app *App) handleTag(w http.ResponseWriter, r *http.Request) {
	tagSlug := r.PathValue("tag")
	tagName, err := models.FindTagBySlug(app.DB, tagSlug)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if tagName == "" {
		app.renderNotFound(w, r)
		return
	}

	now := time.Now().UTC()
	perPage, err := models.PostsPerPage(app.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	page, totalPages, offset := paginate(r, perPage, func() (int, error) {
		return models.CountPostsByTag(app.DB, tagName, now)
	})
	if totalPages == 0 {
		app.renderNotFound(w, r)
		return
	}
	posts, err := models.ListPostsByTagPaged(app.DB, tagName, perPage, offset, now)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	opts := &render.PageOpts{
		Tag:        tagName,
		TagSlug:    tagSlug,
		Page:       page,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	}
	if err := app.Renderer.RenderPage(w, "tag", nil, posts, opts, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (app *App) handlePost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	now := time.Now().UTC()
	post, err := models.GetPostBySlug(app.DB, slug)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if post != nil && models.IsPubliclyVisible(post, now) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := app.Renderer.RenderPage(w, "post", post, nil, nil, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	page, err := models.GetStaticPageBySlug(app.DB, slug)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if page == nil || page.Status != "published" {
		app.renderNotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := app.Renderer.RenderPage(w, "page", nil, nil, nil, page); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (app *App) renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	if err := app.Renderer.RenderPage(w, "not_found", nil, nil, nil, nil); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func paginate(r *http.Request, perPage int, countFn func() (int, error)) (page, totalPages, offset int) {
	page = 1
	if p, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && p > 1 {
		page = p
	}
	total, _ := countFn()
	totalPages = (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset = (page - 1) * perPage
	return page, totalPages, offset
}
