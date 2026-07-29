package presentation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPresentationInfrastructureIsInstanceOwned(t *testing.T) {
	t.Parallel()

	first, err := NewMux()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewMux()
	if err != nil {
		t.Fatal(err)
	}
	if first == nil || second == nil || first == second {
		t.Fatal("NewMux() did not return distinct muxes")
	}
	catalog, err := NewCatalog()
	if err != nil {
		t.Fatal(err)
	}
	renderer, err := NewRenderer(catalog)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	if err := renderer.Render(
		t.Context(),
		response,
		"welcome",
		http.StatusOK,
		struct{ Page Page }{Page: NewPage(catalog, "en", "home")},
	); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "<h1>Welcome</h1>") {
		t.Fatalf("welcome response = %d %s", response.Code, response.Body)
	}
	staticResponse := httptest.NewRecorder()
	first.ServeHTTP(
		staticResponse,
		httptest.NewRequest(http.MethodGet, "/resources/petclinic.css", nil),
	)
	if staticResponse.Code != http.StatusOK ||
		!strings.Contains(staticResponse.Body.String(), "--petclinic-green") {
		t.Fatalf("static response = %d %s", staticResponse.Code, staticResponse.Body)
	}
}
