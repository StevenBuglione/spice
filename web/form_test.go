package web

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/StevenBuglione/spice/validation"
)

func TestBindingResultIsImmutableAndSupportsRejection(t *testing.T) {
	t.Parallel()

	initial, err := validation.Field("owner.city", "required", "is required")
	if err != nil {
		t.Fatal(err)
	}
	result, err := NewBindingResult(initial)
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := result.Reject(
		"owner.telephone",
		"invalid",
		"is invalid",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Errors().Len() != 1 ||
		rejected.Errors().Len() != 2 ||
		result.Valid() ||
		rejected.Valid() {
		t.Fatalf(
			"result=%#v rejected=%#v",
			result.Errors().All(),
			rejected.Errors().All(),
		)
	}
}

func TestBindingResultRejectBindingIsSafe(t *testing.T) {
	t.Parallel()

	result, err := NewBindingResult()
	if err != nil {
		t.Fatal(err)
	}
	result, err = result.RejectBinding(
		NewBindingError(
			LocationForm,
			"age",
			"must be an integer",
			errors.New("raw parser failure"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	violations := result.Errors().All()
	if len(violations) != 1 ||
		violations[0].Field != "age" ||
		violations[0].Code != "binding.invalid" ||
		strings.Contains(violations[0].Message, "raw parser failure") {
		t.Fatalf("form violations = %#v", violations)
	}

	generic, err := result.RejectBinding(
		errors.New("database password secret"),
	)
	if err != nil {
		t.Fatal(err)
	}
	all := generic.Errors().All()
	if len(all) != 2 ||
		all[1].Field != "" ||
		all[1].Message != "the submitted form is invalid" {
		t.Fatalf("generic violations = %#v", all)
	}
}

func TestDecodeFormIsBoundedStrictAndDoesNotMutateRequest(t *testing.T) {
	t.Parallel()

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/owners/new",
		strings.NewReader("firstName=Joe&lastName=Bloggs"),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded; charset=UTF-8",
	)
	values, err := DecodeForm(request, 128)
	if err != nil {
		t.Fatal(err)
	}
	if values.Get("firstName") != "Joe" ||
		values.Get("lastName") != "Bloggs" ||
		request.Form != nil {
		t.Fatalf("values=%#v request.Form=%#v", values, request.Form)
	}
	firstNames := values["firstName"]
	if len(firstNames) != 1 {
		t.Fatalf("firstName values = %#v", firstNames)
	}
	firstNames[0] = "changed"
	if request.Form != nil {
		t.Fatal("DecodeForm mutated request.Form")
	}
	if err := RejectUnknownForm(values, []string{
		"firstName",
		"lastName",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeFormAndFieldValidationFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		request     *http.Request
		maximum     int64
		wantProblem string
	}{
		{name: "nil request", wantProblem: "form"},
		{
			name: "wrong media type",
			request: formRequest(
				t,
				"application/json",
				[]byte("{}"),
			),
			wantProblem: "Content-Type",
		},
		{
			name: "wrong charset",
			request: formRequest(
				t,
				"application/x-www-form-urlencoded; charset=iso-8859-1",
				[]byte("name=Joe"),
			),
			wantProblem: "UTF-8",
		},
		{
			name: "too large",
			request: formRequest(
				t,
				formMediaType,
				[]byte("name=Joe"),
			),
			maximum:     4,
			wantProblem: "byte limit",
		},
		{
			name: "invalid encoding",
			request: formRequest(
				t,
				formMediaType,
				[]byte("%zz"),
			),
			wantProblem: "URL-encoded",
		},
	}
	for _, test := range tests {
		_, err := DecodeForm(test.request, test.maximum)
		if err == nil || !strings.Contains(err.Error(), test.wantProblem) {
			t.Fatalf("%s: DecodeForm() error = %v", test.name, err)
		}
	}
	values := url.Values{"name": {"one", "two"}}
	if _, _, err := FormValue(values, "name", true); err == nil {
		t.Fatal("FormValue(repeated) error = nil")
	}
	if err := RejectUnknownForm(
		url.Values{"admin": {"true"}},
		[]string{"name"},
	); err == nil || strings.Contains(err.Error(), "true") {
		t.Fatalf("RejectUnknownForm() error = %v", err)
	}
	if err := RejectUnknownForm(nil, []string{"name", "name"}); err == nil {
		t.Fatal("RejectUnknownForm(duplicate allowlist) error = nil")
	}
}

func FuzzDecodeForm(f *testing.F) {
	f.Add([]byte("name=Joe"))
	f.Add([]byte("%zz"))
	f.Fuzz(func(t *testing.T, content []byte) {
		request := formRequest(
			t,
			formMediaType,
			content,
		)
		values, err := DecodeForm(request, 1024)
		if len(content) > 1024 && err == nil {
			t.Fatal("oversized form succeeded")
		}
		if err == nil {
			for name, items := range values {
				if name == "" && len(items) == 0 {
					t.Fatal("invalid empty form entry")
				}
			}
		}
	})
}

func formRequest(
	t *testing.T,
	contentType string,
	content []byte,
) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/",
		bytes.NewReader(content),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", contentType)
	return request
}
