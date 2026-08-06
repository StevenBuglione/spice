package main

import (
	"slices"
	"testing"
)

func TestRepositoryMakefileExposesCrossPlatformFastTarget(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := checkFastTarget(root); err != nil {
		t.Fatal(err)
	}
}

func TestValidateFastTargetAcceptsDirectGoRecipeAcrossLineEndings(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		".PHONY: check fast verify\n\nfast:\n\tgo run ./internal/qualitygate -mode=fast\n",
		".PHONY: check fast verify\r\n\r\nfast:\r\n\tgo run ./internal/qualitygate -mode=fast\r\n",
	} {
		if err := validateFastTarget(content); err != nil {
			t.Fatalf("validateFastTarget() error = %v", err)
		}
	}
}

func TestValidateFastTargetRejectsMissingOrShellSpecificRecipes(t *testing.T) {
	t.Parallel()
	for _, content := range []string{
		".PHONY: check verify\n",
		".PHONY: fast\n\nfast:\n\tbash ./scripts/fast.sh\n",
		"fast:\n\tgo run ./internal/qualitygate -mode=fast\n",
	} {
		if err := validateFastTarget(content); err == nil {
			t.Fatalf("validateFastTarget(%q) succeeded, want error", content)
		}
	}
}

func TestSelectAffectedPackagesIncludesReverseAndTestImportClosure(t *testing.T) {
	t.Parallel()
	qualitygate := modulePath + "/internal/qualitygate"
	packages := []packageMetadata{
		{ImportPath: modulePath + "/bean", Directory: "bean"},
		{ImportPath: modulePath + "/web", Directory: "web", Imports: []string{modulePath + "/bean"}},
		{ImportPath: modulePath + "/spicetest", Directory: "spicetest", TestImports: []string{modulePath + "/bean"}},
		{ImportPath: qualitygate, Directory: "internal/qualitygate"},
	}
	tests := []struct {
		name    string
		changed []string
		want    []string
	}{
		{
			name:    "public package and reverse consumers",
			changed: []string{"bean/bean.go"},
			want: []string{
				modulePath + "/bean",
				modulePath + "/spicetest",
				modulePath + "/web",
			},
		},
		{
			name:    "Windows path",
			changed: []string{`web\web_test.go`},
			want:    []string{modulePath + "/web"},
		},
		{
			name:    "documentation",
			changed: []string{"docs/verification.md"},
			want:    []string{qualitygate},
		},
		{
			name: "clean tree",
			want: []string{qualitygate},
		},
		{
			name:    "module graph",
			changed: []string{"go.mod"},
			want: []string{
				modulePath + "/bean",
				qualitygate,
				modulePath + "/spicetest",
				modulePath + "/web",
			},
		},
		{
			name:    "unknown Go ownership widens",
			changed: []string{"removed/package.go"},
			want: []string{
				modulePath + "/bean",
				qualitygate,
				modulePath + "/spicetest",
				modulePath + "/web",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := selectAffectedPackages(test.changed, packages)
			if !slices.Equal(got, test.want) {
				t.Fatalf("selectAffectedPackages() = %v, want %v", got, test.want)
			}
		})
	}
}
