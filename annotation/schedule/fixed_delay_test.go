package schedule

import "testing"

func TestFixedDelayDefinition(t *testing.T) {
	t.Parallel()
	if err := FixedDelay().Validate(); err != nil {
		t.Fatalf("FixedDelay() definition: %v", err)
	}
}
