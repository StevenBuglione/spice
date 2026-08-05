package presentation

import (
	"context"
	"embed"
	"fmt"
	"net/http"

	"github.com/spice-framework/spice/resource"
)

//go:embed static/*
var staticFiles embed.FS

func staticHandler() (http.Handler, error) {
	loader, err := resource.NewLoader(resource.Mount{
		Name: "petclinic",
		FS:   staticFiles,
	})
	if err != nil {
		return nil, fmt.Errorf("construct Petclinic resource loader: %w", err)
	}
	staticRoot, err := loader.Resolve("spice://petclinic/static")
	if err != nil {
		return nil, fmt.Errorf("resolve Petclinic static resource: %w", err)
	}
	source, err := staticRoot.Sub(context.Background())
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
