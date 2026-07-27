package observability

import "testing"

func TestLoggingDefinition(t *testing.T) {
	t.Parallel()
	if err := Logging().Validate(); err != nil {
		t.Fatalf("Logging() definition: %v", err)
	}
}
