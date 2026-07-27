package management

import "testing"

func TestEnableDefinition(t *testing.T) {
	t.Parallel()
	if err := Enable().Validate(); err != nil {
		t.Fatalf("Enable() definition: %v", err)
	}
}
