package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type packageMetadata struct {
	ImportPath   string
	Directory    string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

func fast(ctx context.Context, root string) error {
	if err := checkGoVersion(); err != nil {
		return err
	}
	if err := checkFastTarget(root); err != nil {
		return err
	}
	packages, err := loadPackageMetadata(ctx, root)
	if err != nil {
		return err
	}
	changed, err := changedRepositoryFiles(ctx, root)
	if err != nil {
		return err
	}
	affected := selectAffectedPackages(changed, packages)
	if len(affected) == 0 {
		return errors.New("fast feedback selected no packages")
	}
	if _, err := fmt.Fprintf(os.Stdout, "fast feedback: %d package(s): %s\n", len(affected), strings.Join(affected, ", ")); err != nil {
		return fmt.Errorf("write fast-feedback selection: %w", err)
	}
	arguments := []string{"test", "-mod=readonly", "-count=1"}
	arguments = append(arguments, affected...)
	return runGo(ctx, root, offlineEnvironment(), arguments...)
}

func checkFastTarget(root string) error {
	content, err := os.ReadFile(filepath.Join(root, "Makefile")) // #nosec G304 -- root is repository-owned.
	if err != nil {
		return fmt.Errorf("read Makefile fast-feedback contract: %w", err)
	}
	return validateFastTarget(string(content))
}

func validateFastTarget(content string) error {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	phony := false
	recipe := false
	for index, line := range lines {
		if strings.HasPrefix(line, ".PHONY:") && slices.Contains(strings.Fields(strings.TrimPrefix(line, ".PHONY:")), "fast") {
			phony = true
		}
		if line == "fast:" && index+1 < len(lines) && lines[index+1] == "\tgo run ./internal/qualitygate -mode=fast" {
			recipe = true
		}
	}
	if !phony || !recipe {
		return errors.New("makefile must declare a phony fast target that directly runs the Go quality orchestrator")
	}
	return nil
}

func changedRepositoryFiles(ctx context.Context, root string) ([]string, error) {
	tracked, err := capture(ctx, root, nil, "git", "diff", "--name-only", "--diff-filter=ACDMRTUXB", "-z", "HEAD", "--")
	if err != nil {
		return nil, fmt.Errorf("list changed tracked files: %w", err)
	}
	untracked, err := capture(ctx, root, nil, "git", "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("list untracked files: %w", err)
	}
	paths := make(map[string]struct{})
	for _, output := range []string{tracked, untracked} {
		for path := range strings.SplitSeq(output, "\x00") {
			normalized := normalizeRepositoryPath(path)
			if normalized == "" || strings.HasPrefix(normalized, ".tmp/") {
				continue
			}
			paths[normalized] = struct{}{}
		}
	}
	result := slices.Collect(maps.Keys(paths))
	sort.Strings(result)
	return result, nil
}

func loadPackageMetadata(ctx context.Context, root string) ([]packageMetadata, error) {
	stdout, err := captureGo(ctx, root, offlineEnvironment(), "list", "-mod=readonly", "-json", "./...")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(strings.NewReader(stdout))
	packages := make([]packageMetadata, 0, expectedPublicPackageCount+1)
	for {
		var listed struct {
			ImportPath   string
			Dir          string
			Imports      []string
			TestImports  []string
			XTestImports []string
		}
		if err := decoder.Decode(&listed); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode go list package metadata: %w", err)
		}
		if listed.ImportPath != modulePath && !strings.HasPrefix(listed.ImportPath, modulePath+"/") {
			continue
		}
		relative, err := filepath.Rel(root, listed.Dir)
		if err != nil {
			return nil, fmt.Errorf("relativize package %s: %w", listed.ImportPath, err)
		}
		packages = append(packages, packageMetadata{
			ImportPath:   listed.ImportPath,
			Directory:    normalizeRepositoryPath(relative),
			Imports:      listed.Imports,
			TestImports:  listed.TestImports,
			XTestImports: listed.XTestImports,
		})
	}
	sort.Slice(packages, func(left, right int) bool { return packages[left].ImportPath < packages[right].ImportPath })
	return packages, nil
}

func selectAffectedPackages(changed []string, packages []packageMetadata) []string {
	qualitygate := modulePath + "/internal/qualitygate"
	byDirectory := make(map[string]string, len(packages))
	selected := make(map[string]struct{})
	for _, current := range packages {
		byDirectory[normalizeRepositoryPath(current.Directory)] = current.ImportPath
	}
	if len(changed) == 0 {
		selected[qualitygate] = struct{}{}
	}
	for _, path := range changed {
		normalized := normalizeRepositoryPath(path)
		if normalized == "go.mod" || normalized == "go.sum" {
			return allPackagePaths(packages)
		}
		if packagePath, found := packageForChangedFile(normalized, byDirectory); found {
			selected[packagePath] = struct{}{}
			continue
		}
		if strings.HasSuffix(normalized, ".go") {
			return allPackagePaths(packages)
		}
		selected[qualitygate] = struct{}{}
	}
	for changedSelection := true; changedSelection; {
		changedSelection = false
		for _, current := range packages {
			if _, exists := selected[current.ImportPath]; exists {
				continue
			}
			dependencies := append(slices.Clone(current.Imports), current.TestImports...)
			dependencies = append(dependencies, current.XTestImports...)
			for _, dependency := range dependencies {
				if _, affected := selected[dependency]; affected {
					selected[current.ImportPath] = struct{}{}
					changedSelection = true
					break
				}
			}
		}
	}
	result := slices.Collect(maps.Keys(selected))
	sort.Strings(result)
	return result
}

func packageForChangedFile(path string, byDirectory map[string]string) (string, bool) {
	directory := normalizeRepositoryPath(filepath.Dir(filepath.FromSlash(path)))
	for {
		if packagePath, found := byDirectory[directory]; found {
			return packagePath, true
		}
		if directory == "." || directory == "" {
			return "", false
		}
		directory = normalizeRepositoryPath(filepath.Dir(filepath.FromSlash(directory)))
	}
}

func allPackagePaths(packages []packageMetadata) []string {
	result := make([]string, 0, len(packages))
	for _, current := range packages {
		result = append(result, current.ImportPath)
	}
	sort.Strings(result)
	return result
}

func normalizeRepositoryPath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	normalized = strings.TrimPrefix(normalized, "./")
	if normalized == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(normalized)))
}
