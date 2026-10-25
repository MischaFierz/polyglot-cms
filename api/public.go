package main

import (
	"errors"
	"net/http"
	"strconv"
)

// Settings that may be edited through the API. Anything else is rejected.
var editableSettings = map[string]bool{
	"site_title": true, "tagline": true, "footer": true, "home_slug": true, "posts_per_page": true,
}

func (a *App) loadSettings() (map[string]string, error) {
	rows, err := a.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, rows.Err()
}

func (a *App) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := a.loadSettings()
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (a *App) putSettings(w http.ResponseWriter, r *http.Request) {
	var in map[string]string
	if !readJSON(w, r, &in) {
		return
	}
	for k := range in {
		if !editableSettings[k] {
			writeError(w, http.StatusUnprocessableEntity, "unknown setting: "+k)
			return
		}
	}
	if v, ok := in["posts_per_page"]; ok {
		if n, err := strconv.Atoi(v); err != nil || n < 1 || n > 100 {
			writeError(w, http.StatusUnprocessableEntity, "posts_per_page must be between 1 and 100")
			return
		}
	}
	if v, ok := in["home_slug"]; ok {
		in["home_slug"] = slugify(v)
	}

	tx, err := a.db.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()
	for k, v := range in {
		if _, err := tx.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
			ON CONFLICT (key) DO UPDATE SET value = excluded.value`, k, v); err != nil {
			serverError(w, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}
	a.getSettings(w, r)
}

type navItem struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

func (a *App) publicSite(w http.ResponseWriter, r *http.Request) {
	settings, err := a.loadSettings()
	if err != nil {
		serverError(w, err)
		return
	}
	rows, err := a.db.Query(`SELECT title, slug FROM pages
		WHERE status = 'published' AND show_in_nav = 1 ORDER BY nav_order, title`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	nav := []navItem{}
	for rows.Next() {
		var item navItem
		if err := rows.Scan(&item.Title, &item.Slug); err != nil {
			serverError(w, err)
			return
		}
		nav = append(nav, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings, "nav": nav})
}

func (a *App) publicPage(w http.ResponseWriter, r *http.Request) {
	p, err := scanPage(a.db.QueryRow(pageSelect+" WHERE p.slug = ? AND p.status = 'published'", r.PathValue("slug")))
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	p.BodyMD = ""
	writeJSON(w, http.StatusOK, p)
}

func (a *App) publicPosts(w http.ResponseWriter, r *http.Request) {
	settings, err := a.loadSettings()
	if err != nil {
		serverError(w, err)
		return
	}
	perPage, err := strconv.Atoi(settings["posts_per_page"])
	if err != nil {
		perPage = 10
	}
	perPage = queryInt(r, "limit", perPage, 1, 100)
	page := queryInt(r, "page", 1, 1, 100000)

	var total int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM pages WHERE type = 'post' AND status = 'published'`).Scan(&total); err != nil {
		serverError(w, err)
		return
	}
	posts, err := a.queryPages(`WHERE p.type = 'post' AND p.status = 'published'
		ORDER BY p.published_at DESC LIMIT ? OFFSET ?`, perPage, (page-1)*perPage)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"posts":       withoutBodies(posts),
		"page":        page,
		"total_pages": (total + perPage - 1) / perPage,
		"total":       total,
	})
}

func (a *App) publicSitemap(w http.ResponseWriter, r *http.Request) {
	pages, err := a.queryPages(`WHERE p.status = 'published' ORDER BY p.type, p.slug`)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, withoutBodies(pages))
}
