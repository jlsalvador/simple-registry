package handler

import (
	"embed"
	"net/http"
)

//go:embed web/*
var webFS embed.FS

var webHandler = http.FileServer(http.FS(webFS))

func (m *ServeMux) Web(
	w http.ResponseWriter,
	r *http.Request,
) {
	webHandler.ServeHTTP(w, r)
}
