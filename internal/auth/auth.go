package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"time"

	"slop/internal/models"
)

const (
	cookieName    = "slop_session"
	sessionMaxAge = 30 * 24 * time.Hour
)

type ctxKey int

const userCtxKey ctxKey = 0

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Login validates credentials and, on success, creates a session and sets the cookie.
func Login(w http.ResponseWriter, r *http.Request, database *sql.DB, username, password string) (bool, error) {
	user, err := models.GetUserByUsername(database, username)
	if err != nil {
		return false, err
	}
	if user == nil || !CheckPassword(user.PasswordHash, password) {
		return false, nil
	}
	token, err := newToken()
	if err != nil {
		return false, err
	}
	expires := time.Now().Add(sessionMaxAge)
	if err := models.CreateSession(database, token, user.ID, expires); err != nil {
		return false, err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return true, nil
}

// Logout deletes the current session and clears the cookie.
func Logout(w http.ResponseWriter, r *http.Request, database *sql.DB) error {
	if c, err := r.Cookie(cookieName); err == nil {
		_ = models.DeleteSession(database, c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// CurrentUser returns the authenticated user for the request, or nil.
func CurrentUser(r *http.Request, database *sql.DB) (*models.User, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil, nil
	}
	sess, err := models.GetSession(database, c.Value)
	if err != nil || sess == nil {
		return nil, err
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = models.DeleteSession(database, sess.Token)
		return nil, nil
	}
	return models.GetUserByID(database, sess.UserID)
}

// RequireAuth wraps a handler, redirecting unauthenticated users to the login page.
func RequireAuth(database *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := CurrentUser(r, database)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		// Force a credential change on first login before any other admin action.
		if user.MustChange && r.URL.Path != "/admin/account" && r.URL.Path != "/admin/logout" {
			http.Redirect(w, r, "/admin/account", http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), userCtxKey, user)
		next(w, r.WithContext(ctx))
	}
}

// SetUserContext attaches the authenticated user to a request context.
func SetUserContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userCtxKey, user)
}

// UserFromContext returns the user placed by RequireAuth.
func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(userCtxKey).(*models.User)
	return u
}
