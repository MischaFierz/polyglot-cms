// Command api is the headless content API of Polyglot CMS. It owns the
// database, authentication, Markdown rendering and media uploads, and serves
// the compiled admin panel.
package main

import (
	"database/sql"
	"embed"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/yuin/goldmark"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type App struct {
	db        *sql.DB
	md        goldmark.Markdown
	uploadDir string
}

func main() {
	app := &App{
		md:        newMarkdown(),
		uploadDir: env("CMS_UPLOAD_DIR", "../site/public/uploads"),
	}

	db, err := openDB(env("CMS_DB", "cms.db"))
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app.db = db

	if err := migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	if err := app.seedAdmin(env("CMS_ADMIN_EMAIL", "admin@example.com"), env("CMS_ADMIN_PASSWORD", "admin")); err != nil {
		log.Fatalf("seed admin: %v", err)
	}
	if err := app.renderMissingHTML(); err != nil {
		log.Fatalf("render pages: %v", err)
	}
	if err := os.MkdirAll(app.uploadDir, 0o755); err != nil {
		log.Fatalf("create upload dir: %v", err)
	}

	addr := env("CMS_ADDR", ":8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           withLogging(app.routes(env("CMS_ADMIN_DIR", "../admin"))),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("API listening on %s (admin panel at /admin/)", addr)
	log.Fatal(srv.ListenAndServe())
}

func (a *App) routes(adminDir string) http.Handler {
	mux := http.NewServeMux()

	// Authentication
	mux.HandleFunc("POST /api/auth/login", a.login)
	mux.HandleFunc("POST /api/auth/logout", a.auth(a.logout))
	mux.HandleFunc("GET /api/me", a.auth(a.me))

	// Content management
	mux.HandleFunc("GET /api/pages", a.auth(a.listPages))
	mux.HandleFunc("POST /api/pages", a.auth(a.createPage))
	mux.HandleFunc("GET /api/pages/{id}", a.auth(a.getPage))
	mux.HandleFunc("PUT /api/pages/{id}", a.auth(a.updatePage))
	mux.HandleFunc("DELETE /api/pages/{id}", a.auth(a.deletePage))
	mux.HandleFunc("GET /api/pages/{id}/revisions", a.auth(a.listRevisions))
	mux.HandleFunc("POST /api/preview", a.auth(a.preview))

	mux.HandleFunc("GET /api/media", a.auth(a.listMedia))
	mux.HandleFunc("POST /api/media", a.auth(a.uploadMedia))
	mux.HandleFunc("DELETE /api/media/{id}", a.auth(a.deleteMedia))

	mux.HandleFunc("GET /api/settings", a.auth(a.getSettings))
	mux.HandleFunc("PUT /api/settings", a.auth(a.requireAdmin(a.putSettings)))

	// Public, read-only endpoints used by the website renderer
	mux.HandleFunc("GET /api/public/site", a.publicSite)
	mux.HandleFunc("GET /api/public/pages/{slug}", a.publicPage)
	mux.HandleFunc("GET /api/public/posts", a.publicPosts)
	mux.HandleFunc("GET /api/public/sitemap", a.publicSitemap)

	// Admin panel (static files) and uploaded media
	mux.Handle("GET /admin/", http.StripPrefix("/admin/", http.FileServer(http.Dir(adminDir))))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(a.uploadDir))))
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})

	return mux
}
