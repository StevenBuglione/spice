package coretool

import "testing"

func TestPathIsFullyQualifiedGoToolPackage(t *testing.T) {
	t.Parallel()
	const expected = "github.com/StevenBuglione/spice/cmd/spice-annotation-core"
	if Path != expected {
		t.Fatalf("Path = %q, want %q", Path, expected)
	}
}
