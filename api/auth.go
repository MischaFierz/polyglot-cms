package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionTTL = 7 * 24 * time.Hour

type User struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type userCtxKey struct{}

func currentUser(r *http.Request) *User {
	u, _ := r.Context().Value(userCtxKey{}).(*User)
	return u
}

// seedAdmin creates the first admin account when the users table is empty.
func (a *App) seedAdmin(email, password string) error {
	var count int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count); err != nil || count > 0 {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := a.db.Exec(`INSERT INTO users (email, name, password_hash, role) VALUES (?, 'Admin', ?, 'admin')`,
		email, string(hash)); err != nil {
		return err
	}
	log.Printf("created admin user %s", email)
	if password == "admin" {
		log.Printf("WARNING: default password in use; set CMS_ADMIN_PASSWORD before going live")
	}
	return nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &in) {
		return
	}

	var u User
	var hash string
	err := a.db.QueryRow(`SELECT id, email, name, role, password_hash FROM users WHERE email = ?`,
		strings.TrimSpace(in.Email)).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &hash)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		serverError(w, err)
		return
	}
	token := hex.EncodeToString(raw)
	expires := time.Now().UTC().Add(sessionTTL).Format(time.RFC3339)

	if _, err := a.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now()); err != nil {
		serverError(w, err)
		return
	}
	if _, err := a.db.Exec(`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`,
		hashToken(token), u.ID, expires); err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if _, err := a.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, hashToken(token)); err != nil {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

// auth rejects requests without a valid bearer token and puts the user in the context.
func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}
		var u User
		err := a.db.QueryRow(`
			SELECT u.id, u.email, u.name, u.role
			FROM sessions s JOIN users u ON u.id = s.user_id
			WHERE s.token_hash = ? AND s.expires_at > ?`, hashToken(token), now()).
			Scan(&u.ID, &u.Email, &u.Name, &u.Role)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "session expired")
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userCtxKey{}, &u)))
	}
}

func (a *App) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := currentUser(r); u == nil || u.Role != "admin" {
			writeError(w, http.StatusForbidden, "admin role required")
			return
		}
		next(w, r)
	}
}
