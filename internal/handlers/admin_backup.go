package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"slop/internal/backup"
	"slop/internal/db"
)

func (app *App) handleBackupPage(w http.ResponseWriter, r *http.Request) {
	app.renderAdmin(w, r, "backup", adminCtx{
		Title:  "Backup & Restore",
		Active: "backup",
		Flash:  r.URL.Query().Get("flash"),
		Error:  r.URL.Query().Get("error"),
	})
}

func (app *App) handleBackupDownload(w http.ResponseWriter, r *http.Request) {
	full := r.URL.Query().Get("full") == "1" || r.URL.Query().Get("full") == "true"
	filename := backup.SuggestedFilename(full)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if full {
		w.Header().Set("Content-Type", "application/zip")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	if err := backup.Export(w, app.DB, app.StaticDir, full); err != nil {
		http.Error(w, "export failed: "+err.Error(), http.StatusInternalServerError)
	}
}

func (app *App) handleBackupRestore(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		app.renderBackupError(w, r, "Could not read upload.")
		return
	}
	if r.FormValue("confirm") != "yes" {
		app.renderBackupError(w, r, "You must confirm that you want to replace the current site.")
		return
	}
	file, header, err := r.FormFile("backup")
	if err != nil {
		app.renderBackupError(w, r, "Please choose a backup file (.db or .zip).")
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext != ".db" && ext != ".zip" {
		app.renderBackupError(w, r, "Backup must be a .db or .zip file.")
		return
	}

	tmp, err := os.CreateTemp("", "slop-upload-*"+ext)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, file); err != nil {
		tmp.Close()
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if err := tmp.Close(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	app.dbMu.Lock()
	defer app.dbMu.Unlock()

	if err := app.closeDB(); err != nil {
		app.renderBackupError(w, r, "Could not release database: "+err.Error())
		return
	}

	_, err = backup.Restore(tmpPath, backup.RestoreOptions{
		DBPath:        app.DBPath,
		StaticDir:     app.StaticDir,
		Force:         true,
		SkipLockCheck: true,
	})
	if err != nil {
		_ = app.reopenDB()
		app.renderBackupError(w, r, err.Error())
		return
	}
	if err := app.reopenDB(); err != nil {
		http.Error(w, "restore wrote files but could not reopen database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/backup?flash=Restore+complete", http.StatusSeeOther)
}

func (app *App) renderBackupError(w http.ResponseWriter, r *http.Request, msg string) {
	app.renderAdmin(w, r, "backup", adminCtx{
		Title:  "Backup & Restore",
		Active: "backup",
		Error:  msg,
	})
}

func (app *App) closeDB() error {
	if app.DB == nil {
		return nil
	}
	err := app.DB.Close()
	app.DB = nil
	return err
}

func (app *App) reopenDB() error {
	database, err := db.Open(app.DBPath)
	if err != nil {
		return err
	}
	app.DB = database
	app.Renderer.SetDB(database)
	return nil
}
