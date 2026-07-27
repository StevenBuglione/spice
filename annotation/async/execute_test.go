package async

import "testing"

func TestExecuteDefinition(t *testing.T) {
	t.Parallel()
	if err := Execute().Validate(); err != nil {
		t.Fatalf("Execute() definition: %v", err)
	}
}
