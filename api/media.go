package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const maxUploadSize = 20 << 20

// Allowed upload types, sniffed from file content rather than trusted from the client.
// SVG is deliberately excluded because it can carry scripts.
var allowedMedia = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"application/pdf": ".pdf",
	"video/mp4":       ".mp4",
}

type Media struct {
	ID           int64  `json:"id"`
	URL          string `json:"url"`
	OriginalName string `json:"original_name"`
	Mime         string `json:"mime"`
	Size         int64  `json:"size"`
	Alt          string `json:"alt"`
	CreatedAt    string `json:"created_at"`
}

func (a *App) listMedia(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(`SELECT id, filename, original_name, mime, size, alt, created_at FROM media ORDER BY id DESC`)
	if err != nil {
		serverError(w, err)
		return
	}
	defer rows.Close()
	items := []Media{}
	for rows.Next() {
		var m Media
		var filename string
		if err := rows.Scan(&m.ID, &filename, &m.OriginalName, &m.Mime, &m.Size, &m.Alt, &m.CreatedAt); err != nil {
			serverError(w, err)
			return
		}
		m.URL = "/uploads/" + filename
		items = append(items, m)
	}
	writeJSON(w, http.StatusOK, items)
}

func (a *App) uploadMedia(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize+1<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "expected a multipart field named \"file\" (max 20 MB)")
		return
	}
	defer file.Close()

	sniff := make([]byte, 512)
	n, err := io.ReadFull(file, sniff)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		writeError(w, http.StatusBadRequest, "could not read upload")
		return
	}
	mime := http.DetectContentType(sniff[:n])
	ext, ok := allowedMedia[mime]
	if !ok {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported file type: "+mime)
		return
	}

	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		serverError(w, err)
		return
	}
	filename := hex.EncodeToString(raw) + ext
	dst, err := os.Create(filepath.Join(a.uploadDir, filename))
	if err != nil {
		serverError(w, err)
		return
	}
	size, err := io.Copy(dst, io.MultiReader(strings.NewReader(string(sniff[:n])), file))
	dst.Close()
	if err != nil {
		os.Remove(dst.Name())
		serverError(w, err)
		return
	}

	alt := strings.TrimSpace(r.FormValue("alt"))
	if alt == "" {
		alt = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}
	ts := now()
	res, err := a.db.Exec(`INSERT INTO media (filename, original_name, mime, size, alt, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		filename, header.Filename, mime, size, alt, ts)
	if err != nil {
		os.Remove(dst.Name())
		serverError(w, err)
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, Media{
		ID: id, URL: "/uploads/" + filename, OriginalName: header.Filename,
		Mime: mime, Size: size, Alt: alt, CreatedAt: ts,
	})
}

func (a *App) deleteMedia(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var filename string
	if err := a.db.QueryRow(`DELETE FROM media WHERE id = ? RETURNING filename`, id).Scan(&filename); err != nil {
		writeError(w, http.StatusNotFound, "media not found")
		return
	}
	if err := os.Remove(filepath.Join(a.uploadDir, filename)); err != nil && !errors.Is(err, os.ErrNotExist) {
		serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
