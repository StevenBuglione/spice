package cli

import (
	"strings"
	"testing"
)

func TestRunVerifyRejectsAliasDuplicateBeanOutput(t *testing.T) {
	t.Parallel()
	root := writeGoSource(t, `package sample

type Original struct{}
type Alias = Original
type Distinct Original

// @Bean
func OriginalProvider() Original { panic("must not execute") }

// @Bean
func AliasProvider() Alias { panic("must not execute") }

// @Bean
func DistinctProvider() Distinct { panic("must not execute") }
`)
	code, stdout, stderr := runModule(root, "verify", ".")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "1 provider catalog error(s)") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, expected := range []string{
		"exact type example.com/fixture.Alias",
		"example.com/fixture.AliasProvider",
		"example.com/fixture.OriginalProvider",
	} {
		if !strings.Contains(stderr, expected) {
			t.Fatalf("stderr=%q missing %q", stderr, expected)
		}
	}
	if strings.Contains(stderr, "DistinctProvider") {
		t.Fatalf("distinct named provider incorrectly entered alias duplicate diagnostic: %q", stderr)
	}
}
