package security

import "testing"

func TestAuthorizeDefinition(t *testing.T) {
	t.Parallel()
	if err := Authorize().Validate(); err != nil {
		t.Fatalf("Authorize() definition: %v", err)
	}
}
