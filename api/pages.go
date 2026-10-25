package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

type Page struct {
	ID          int64   `json:"id"`
	Type        string  `json:"type"`
	Title       string  `json:"title"`
	Slug        string  `json:"slug"`
	Excerpt     string  `json:"excerpt"`
	BodyMD      string  `json:"body_md,omitempty"`
	BodyHTML    string  `json:"body_html,omitempty"`
	Status      string  `json:"status"`
	Template    string  `json:"template"`
	ShowInNav   bool    `json:"show_in_nav"`
	NavOrder    int     `json:"nav_order"`
	Author      string  `json:"author"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	PublishedAt *string `json:"published_at"`
}

const pageSelect = `
	SELECT p.id, p.type, p.title, p.slug, p.excerpt, p.body_md, p.body_html, p.status, p.template,
	       p.show_in_nav, p.nav_order, COALESCE(u.name, ''), p.created_at, p.updated_at, p.published_at
	FROM pages p LEFT JOIN users u ON u.id = p.author_id`

type scanner interface{ Scan(dest ...any) error }

func scanPage(row scanner) (*Page, error) {
	var p Page
	var published sql.NullString
	err := row.Scan(&p.ID, &p.Type, &p.Title, &p.Slug, &p.Excerpt, &p.BodyMD, &p.BodyHTML, &p.Status,
		&p.Template, &p.ShowInNav, &p.NavOrder, &p.Author, &p.CreatedAt, &p.UpdatedAt, &published)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errNotFound
	}
	if err != nil {
		return nil, err
	}
	if published.Valid {
		p.PublishedAt = &published.String
	}
	return &p, nil
}

func (a *App) queryPages(where string, args ...any) ([]Page, error) {
	rows, err := a.db.Query(pageSelect+" "+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	pages := []Page{}
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, *p)
	}
	return pages, rows.Err()
}

// withoutBodies strips the heavy fields for list responses.
func withoutBodies(pages []Page) []Page {
	for i := range pages {
		pages[i].BodyMD, pages[i].BodyHTML = "", ""
	}
	return pages
}

func newMarkdown() goldmark.Markdown {
	// Raw HTML in Markdown is dropped (goldmark's safe default), so editors cannot inject scripts.
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

func (a *App) renderMarkdown(src string) (string, error) {
	var buf bytes.Buffer
	if err := a.md.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderMissingHTML fills body_html for rows inserted by SQL migrations.
func (a *App) renderMissingHTML() error {
	pages, err := a.queryPages(`WHERE p.body_html = '' AND p.body_md != ''`)
	if err != nil {
		return err
	}
	for _, p := range pages {
		html, err := a.renderMarkdown(p.BodyMD)
		if err != nil {
			return err
		}
		if _, err := a.db.Exec(`UPDATE pages SET body_html = ? WHERE id = ?`, html, p.ID); err != nil {
			return err
		}
	}
	return nil
}

type pageInput struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Slug      string `json:"slug"`
	Excerpt   string `json:"excerpt"`
	BodyMD    string `json:"body_md"`
	Status    string `json:"status"`
	Template  string `json:"template"`
	ShowInNav bool   `json:"show_in_nav"`
	NavOrder  int    `json:"nav_order"`
}

// The PHP renderer resolves templates by file name, so keep them strictly safe.
var templateName = regexp.MustCompile(`^[a-z0-9_-]{1,40}$`)

func (in *pageInput) normalize() error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return errors.New("title is required")
	}
	in.Slug = slugify(in.Slug)
	if in.Slug == "" {
		in.Slug = slugify(in.Title)
	}
	if in.Slug == "" {
		return errors.New("slug must contain letters or digits")
	}
	if in.Type == "" {
		in.Type = "page"
	}
	if in.Type != "page" && in.Type != "post" {
		return errors.New("type must be page or post")
	}
	if in.Status == "" {
		in.Status = "draft"
	}
	if in.Status != "draft" && in.Status != "published" {
		return errors.New("status must be draft or published")
	}
	if in.Template == "" {
		in.Template = "default"
	}
	if !templateName.MatchString(in.Template) {
		return errors.New("template may only contain a-z, 0-9, - and _")
	}
	in.Excerpt = strings.TrimSpace(in.Excerpt)
	return nil
}

func (a *App) listPages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where, args := []string{"1 = 1"}, []any{}
	if t := q.Get("type"); t != "" {
		where, args = append(where, "p.type = ?"), append(args, t)
	}
	if s := q.Get("status"); s != "" {
		where, args = append(where, "p.status = ?"), append(args, s)
	}
	if term := q.Get("q"); term != "" {
		where, args = append(where, "(p.title LIKE ? OR p.slug LIKE ?)"), append(args, "%"+term+"%", "%"+term+"%")
	}
	pages, err := a.queryPages("WHERE "+strings.Join(where, " AND ")+" ORDER BY p.updated_at DESC", args...)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, withoutBodies(pages))
}

func (a *App) getPage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := scanPage(a.db.QueryRow(pageSelect+" WHERE p.id = ?", id))
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *App) createPage(w http.ResponseWriter, r *http.Request) {
	var in pageInput
	if !readJSON(w, r, &in) {
		return
	}
	if err := in.normalize(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	html, err := a.renderMarkdown(in.BodyMD)
	if err != nil {
		serverError(w, err)
		return
	}

	ts := now()
	var publishedAt any
	if in.Status == "published" {
		publishedAt = ts
	}
	res, err := a.db.Exec(`
		INSERT INTO pages (type, title, slug, excerpt, body_md, body_html, status, template,
		                   show_in_nav, nav_order, author_id, created_at, updated_at, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.Type, in.Title, in.Slug, in.Excerpt, in.BodyMD, html, in.Status, in.Template,
		in.ShowInNav, in.NavOrder, currentUser(r).ID, ts, ts, publishedAt)
	if isUniqueViolation(err) {
		writeError(w, http.StatusConflict, fmt.Sprintf("slug %q is already taken", in.Slug))
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	p, err := scanPage(a.db.QueryRow(pageSelect+" WHERE p.id = ?", id))
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *App) updatePage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in pageInput
	if !readJSON(w, r, &in) {
		return
	}
	if err := in.normalize(); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	html, err := a.renderMarkdown(in.BodyMD)
	if err != nil {
		serverError(w, err)
		return
	}

	tx, err := a.db.Begin()
	if err != nil {
		serverError(w, err)
		return
	}
	defer tx.Rollback()

	// Snapshot the current version before overwriting it.
	res, err := tx.Exec(`
		INSERT INTO page_revisions (page_id, title, body_md, author_id, created_at)
		SELECT id, title, body_md, ?, ? FROM pages WHERE id = ?`, currentUser(r).ID, now(), id)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}

	ts := now()
	_, err = tx.Exec(`
		UPDATE pages SET type = ?, title = ?, slug = ?, excerpt = ?, body_md = ?, body_html = ?,
		       status = ?, template = ?, show_in_nav = ?, nav_order = ?, updated_at = ?,
		       published_at = CASE WHEN ? = 'published' THEN COALESCE(published_at, ?) ELSE published_at END
		WHERE id = ?`,
		in.Type, in.Title, in.Slug, in.Excerpt, in.BodyMD, html, in.Status, in.Template,
		in.ShowInNav, in.NavOrder, ts, in.Status, ts, id)
	if isUniqueViolation(err) {
		writeError(w, http.StatusConflict, fmt.Sprintf("slug %q is already taken", in.Slug))
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	// Keep the 50 most recent revisions per page.
	if _, err := tx.Exec(`
		DELETE FROM page_revisions WHERE page_id = ? AND id NOT IN (
			SELECT id FROM page_revisions WHERE page_id = ? ORDER BY id DESC LIMIT 50)`, id, id); err != nil {
		serverError(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		serverError(w, err)
		return
	}

	p, err := scanPage(a.db.QueryRow(pageSelect+" WHERE p.id = ?", id))
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *App) deletePage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	res, err := a.db.Exec(`DELETE FROM pages WHERE id = ?`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "page not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type Revision struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	BodyMD    string `json:"body_md"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

func (a *App) listRevisions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	rows, err := a.db.Query(`
		SELECT r.id, r.title, r.body_md, COALESCE(u.name, ''), r.created_at
		FROM page_revisions r LEFT JOIN users u ON u.id = r.author_id
		WHERE r.page_id = ? ORDER BY r.id DESC`, id)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	revs := []Revision{}
	for rows.Next() {
		var rev Revision
		if err := rows.Scan(&rev.ID, &rev.Title, &rev.BodyMD, &rev.Author, &rev.CreatedAt); err != nil {
			serverError(w, err)
			return
		}
		revs = append(revs, rev)
	}
	writeJSON(w, http.StatusOK, revs)
}

func (a *App) preview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BodyMD string `json:"body_md"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	html, err := a.renderMarkdown(in.BodyMD)
	if err != nil {
		serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"html": html})
}
