package data

import "testing"

func TestTransactionalDefinition(t *testing.T) {
	t.Parallel()
	if err := Transactional().Validate(); err != nil {
		t.Fatalf("Transactional() definition: %v", err)
	}
}
