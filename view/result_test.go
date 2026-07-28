package view

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestResultRendersAndRedirects(t *testing.T) {
	t.Parallel()

	renderer, err := Parse(
		fstest.MapFS{
			"owner.html": {
				Data: []byte(`{{define "owner"}}<h1>{{.Name}}</h1>{{end}}`),
			},
		},
		[]string{"*.html"},
		nil,
		Options{},
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Render("owner", struct{ Name string }{Name: "<Joe>"})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	respondErr := renderer.Respond(
		context.Background(),
		response,
		result,
	)
	if respondErr != nil {
		t.Fatal(respondErr)
	}
	if response.Code != http.StatusOK ||
		response.Body.String() != "<h1>&lt;Joe&gt;</h1>" {
		t.Fatalf("render response = %d %q", response.Code, response.Body)
	}

	redirect, err := SeeOther("/owners/1?created=true")
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	respondErr = renderer.Respond(
		context.Background(),
		response,
		redirect,
	)
	if respondErr != nil {
		t.Fatal(respondErr)
	}
	if response.Code != http.StatusSeeOther ||
		response.Header().Get("Location") != "/owners/1?created=true" ||
		response.Body.Len() != 0 {
		t.Fatalf(
			"redirect response = %d %#v %q",
			response.Code,
			response.Header(),
			response.Body,
		)
	}
}

func TestResultRejectsUnsafeOutcomes(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"", " owner", "owner "} {
		if _, err := Render(name, struct{}{}); err == nil {
			t.Fatalf("Render(%q) error = nil", name)
		}
	}
	if _, err := RenderStatus(
		http.StatusNoContent,
		"owner",
		struct{}{},
	); err == nil {
		t.Fatal("RenderStatus(204) error = nil")
	}
	for _, location := range []string{
		"",
		"relative",
		"//example.test",
		"https://example.test",
		"/owners\r\nInjected: true",
		"/owners#fragment",
	} {
		if _, err := SeeOther(location); err == nil {
			t.Fatalf("SeeOther(%q) error = nil", location)
		}
	}
	response := httptest.NewRecorder()
	if err := (&Renderer{}).Respond(
		context.Background(),
		response,
		Result{},
	); err == nil || !strings.Contains(err.Error(), "uninitialized") {
		t.Fatalf("Respond(zero) error = %v", err)
	}
}

func TestAcceptsHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		header string
		want   bool
	}{
		{"", true},
		{"text/html", true},
		{"text/*;q=0.5", true},
		{"application/xhtml+xml", true},
		{"application/json", false},
		{"text/html;q=0", false},
		{"not a media type", false},
	}
	for _, test := range tests {
		if got := AcceptsHTML(test.header); got != test.want {
			t.Errorf(
				"AcceptsHTML(%q) = %t, want %t",
				test.header,
				got,
				test.want,
			)
		}
	}
}
