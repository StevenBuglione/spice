package system

import "testing"

func TestWelcomeControllerRendersCanonicalPage(t *testing.T) {
	t.Parallel()

	controller := NewWelcomeController()
	result, err := controller.Show(t.Context(), WelcomeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("result validation: %v", err)
	}
}
