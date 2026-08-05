package mysql_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spice-framework/spice/config"
	mysqlgen "github.com/spice-framework/spice/examples/petclinic/internal/spicegen/mysql"
)

type ApplicationOptions = mysqlgen.ApplicationOptions

var (
	NewApplication            = mysqlgen.NewApplication
	NewApplicationWithOptions = mysqlgen.NewApplicationWithOptions
)

func TestGeneratedMySQLGraphUsesMySQLRepositories(
	t *testing.T,
) {
	t.Parallel()

	source, err := config.NewMapSource("mysql-test", map[string]string{
		"petclinic.datasource.url": "mysql://petclinic:petclinic@" +
			"127.0.0.1:1/petclinic?tls=disable",
		"petclinic.datasource.allow-insecure": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	application, err := NewApplicationWithOptions(
		t.Context(),
		ApplicationOptions{Sources: []config.Source{source}},
	)
	if err != nil {
		t.Fatalf("NewApplicationWithOptions() error = %v", err)
	}
	t.Cleanup(func() {
		if stopErr := application.Stop(t.Context()); stopErr != nil {
			t.Errorf("Stop() error = %v", stopErr)
		}
	})

	components := application.Components()
	if components.OpenDatabase == nil ||
		components.OwnerRepository == nil ||
		components.PetTypeRepository == nil ||
		components.VetRepository == nil {
		t.Fatalf("MySQL components = %#v", components)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	handler := application.Handler()
	if handler == nil {
		t.Fatal("Handler() = nil")
	}
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK ||
		!strings.Contains(response.Body.String(), "Petclinic") {
		t.Fatalf(
			"welcome response = %d %q",
			response.Code,
			response.Body.String(),
		)
	}
}

func TestGeneratedMySQLGraphRequiresDatasourceURL(
	t *testing.T,
) {
	t.Parallel()

	application, err := NewApplication(t.Context())
	if err == nil || application != nil {
		t.Fatalf("NewApplication() = %#v, %v", application, err)
	}
	if !strings.Contains(err.Error(), "petclinic.datasource.url") {
		t.Fatalf("NewApplication() error = %v", err)
	}
}
