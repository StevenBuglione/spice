package cache

import "testing"

func TestCacheableDefinition(t *testing.T) {
	t.Parallel()
	if err := Cacheable().Validate(); err != nil {
		t.Fatalf("Cacheable() definition: %v", err)
	}
}
