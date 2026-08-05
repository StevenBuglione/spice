package coretool

import "testing"

func TestPathIsFullyQualifiedGoToolPackage(t *testing.T) {
	t.Parallel()
	const expected = "github.com/spice-framework/toolchain/cmd/spice-annotation-core"
	if Path != expected {
		t.Fatalf("Path = %q, want %q", Path, expected)
	}
}
