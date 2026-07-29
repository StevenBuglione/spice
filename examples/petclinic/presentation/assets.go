package presentation

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

func staticHandler() (http.Handler, error) {
	source, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, fmt.Errorf("construct Petclinic static filesystem: %w", err)
	}
	files := http.FileServer(http.FS(source))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "public, max-age=3600")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		files.ServeHTTP(writer, request)
	}), nil
}
