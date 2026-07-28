package presentation

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPresentationInfrastructureIsInstanceOwned(t *testing.T) {
	t.Parallel()

	first := NewMux()
	second := NewMux()
	if first == nil || second == nil || first == second {
		t.Fatal("NewMux() did not return distinct muxes")
	}
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	if err := renderer.Render(
		t.Context(),
		response,
		"welcome",
		http.StatusOK,
		struct{}{},
	); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "<h1>Welcome</h1>") {
		t.Fatalf("welcome response = %d %s", response.Code, response.Body)
	}
}
