package handler

import (
	"embed"
	"net/http"
	"strings"
)

//go:embed ui/*
var uiFS embed.FS

func (m *ServeMux) RedirectToUI(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui", http.StatusSeeOther)
}

func (m *ServeMux) UI(w http.ResponseWriter, r *http.Request) {
	// 1. Clean the path to match the embed FS structure.
	// If the handler is served at "/ui/", we need to be careful with the prefix.
	// Assuming the request path is /ui/css/style.css, we want ui/css/style.css
	path := strings.TrimPrefix(r.URL.Path, "/")

	// If path is empty (root), set it to index.html to avoid directory listing issues
	if path == "ui" || path == "ui/" {
		path = "ui/index.html"
	}

	// 2. Try to locate the file in the embedded filesystem.
	// Since embed.FS is in-memory, this is very fast.
	f, err := uiFS.Open(path)

	// If the file is not found, or there is an error opening it,
	// it means it's a client-side route (SPA). Serve index.html.
	if err != nil {
		serveIndex(w)
		return
	}

	// We need to check if it's a directory. If so, fallback to index.html
	// to avoid 403 Forbidden or directory listings.
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		f.Close()
		serveIndex(w)
		return
	}

	// 3. The file exists and is a regular file. Serve it.
	f.Close() // Close it, http.FileServer will open it again correctly.

	// Use FileServerFS to handle content-type, ranges, etags, etc. automatically.
	http.FileServer(http.FS(uiFS)).ServeHTTP(w, r)
}

func serveIndex(w http.ResponseWriter) {
	// Read the index.html content directly from the embed FS.
	content, err := uiFS.ReadFile("ui/index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusInternalServerError)
		return
	}

	// Manually set the content type to avoid sniffing issues.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}
