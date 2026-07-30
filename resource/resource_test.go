package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"
)

func TestLoaderResolvesAndReadsMountedResource(t *testing.T) {
	t.Parallel()

	loader, err := NewLoader(
		Mount{
			Name: "assets",
			FS: fstest.MapFS{
				"templates/order.html": {
					Data: []byte("<h1>Order</h1>"),
				},
			},
		},
		Mount{
			Name: "messages",
			FS: fstest.MapFS{
				"en.properties": {Data: []byte("title=Orders")},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := loader.Mounts(); !slices.Equal(
		got,
		[]string{"assets", "messages"},
	) {
		t.Fatalf("Mounts() = %v", got)
	}
	resolved, err := loader.Resolve(
		"spice://assets/templates/order.html",
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Location() != "spice://assets/templates/order.html" ||
		resolved.Mount() != "assets" ||
		resolved.Path() != "templates/order.html" {
		t.Fatalf("Resolve() = %+v", resolved)
	}
	info, err := resolved.Stat(context.Background())
	if err != nil || info.Size() != int64(len("<h1>Order</h1>")) {
		t.Fatalf("Stat() = %+v, %v", info, err)
	}
	content, err := resolved.ReadAll(context.Background(), 1024)
	if err != nil || string(content) != "<h1>Order</h1>" {
		t.Fatalf("ReadAll() = %q, %v", content, err)
	}
	file, err := resolved.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := file.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})
	opened, err := io.ReadAll(file)
	if err != nil || string(opened) != string(content) {
		t.Fatalf("Open() content = %q, %v", opened, err)
	}
}

func TestLoaderRejectsInvalidMountsAndLocations(t *testing.T) {
	t.Parallel()

	if _, err := NewLoader(Mount{Name: "Bad", FS: fstest.MapFS{}}); err == nil {
		t.Fatal("NewLoader() accepted an invalid mount")
	}
	if _, err := NewLoader(Mount{Name: "assets"}); err == nil {
		t.Fatal("NewLoader() accepted a nil filesystem")
	}
	if _, err := NewLoader(
		Mount{Name: "assets", FS: fstest.MapFS{}},
		Mount{Name: "assets", FS: fstest.MapFS{}},
	); err == nil {
		t.Fatal("NewLoader() accepted duplicate mounts")
	}
	loader, err := NewLoader(Mount{Name: "assets", FS: fstest.MapFS{}})
	if err != nil {
		t.Fatal(err)
	}
	locations := []string{
		"",
		"file:///tmp/order.html",
		"spice://assets",
		"spice://assets/",
		"spice://assets/../secret",
		"spice://assets/path?query=true",
		"spice://assets/path#fragment",
		"spice://unknown/path",
		"spice://assets/a%2Fb",
	}
	for _, location := range locations {
		if _, err := loader.Resolve(location); err == nil {
			t.Fatalf("Resolve(%q) succeeded", location)
		}
	}
	var nilLoader *Loader
	if _, err := nilLoader.Resolve("spice://assets/path"); err == nil {
		t.Fatal("nil Loader.Resolve() succeeded")
	}
}

func TestResourceCancellationLimitsAndFailures(t *testing.T) {
	t.Parallel()

	loader, constructionErr := NewLoader(Mount{
		Name: "assets",
		FS: fstest.MapFS{
			"large.txt": {Data: []byte("0123456789")},
			"dir/file":  {Data: []byte("value")},
		},
	})
	if constructionErr != nil {
		t.Fatal(constructionErr)
	}
	resource, resolutionErr := loader.Resolve("spice://assets/large.txt")
	if resolutionErr != nil {
		t.Fatal(resolutionErr)
	}
	if _, err := resource.ReadAll(context.Background(), 5); err == nil {
		t.Fatal("ReadAll() accepted oversized content")
	}
	if _, err := resource.ReadAll(context.Background(), 0); err == nil {
		t.Fatal("ReadAll() accepted a zero limit")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resource.Open(cancelled); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("Open(cancelled) error = %v", err)
	}
	if _, err := resource.Stat(cancelled); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("Stat(cancelled) error = %v", err)
	}
	if _, err := resource.ReadAll(cancelled, 100); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("ReadAll(cancelled) error = %v", err)
	}
	//nolint:staticcheck // The public API promises a fail-closed nil boundary.
	if _, err := resource.Open(nil); err == nil {
		t.Fatal("Open(nil) succeeded")
	}
	var zero Resource
	if _, err := zero.Open(context.Background()); err == nil {
		t.Fatal("zero Resource.Open() succeeded")
	}
	directory, directoryErr := loader.Resolve("spice://assets/dir")
	if directoryErr != nil {
		t.Fatal(directoryErr)
	}
	sub, subErr := directory.Sub(context.Background())
	if subErr != nil {
		t.Fatal(subErr)
	}
	subContent, readErr := fs.ReadFile(sub, "file")
	if readErr != nil || string(subContent) != "value" {
		t.Fatalf("Sub() content = %q, %v", subContent, readErr)
	}
	if _, err := directory.ReadAll(context.Background(), 100); err == nil {
		t.Fatal("ReadAll() accepted a directory")
	}
	if _, err := resource.Sub(context.Background()); err == nil {
		t.Fatal("Sub() accepted a file")
	}
	missing, missingErr := loader.Resolve("spice://assets/missing")
	if missingErr != nil {
		t.Fatal(missingErr)
	}
	if _, err := missing.Open(context.Background()); !errors.Is(
		err,
		fs.ErrNotExist,
	) {
		t.Fatalf("Open(missing) error = %v", err)
	}
}

type cancelReader struct {
	cancel context.CancelFunc
	read   bool
}

func (reader *cancelReader) Read(buffer []byte) (int, error) {
	if reader.read {
		return 0, io.EOF
	}
	reader.read = true
	copy(buffer, "first")
	reader.cancel()
	return len("first"), nil
}

func TestContextReaderStopsBetweenReads(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	reader := &cancelReader{cancel: cancel}
	content, err := io.ReadAll(contextReader{
		cause:  func() error { return context.Cause(ctx) },
		reader: reader,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(content) != "first" {
		t.Fatalf("ReadAll() content = %q", content)
	}
}

func ExampleLoader() {
	loader, err := NewLoader(Mount{
		Name: "assets",
		FS: fstest.MapFS{
			"messages/welcome.txt": {Data: []byte("Welcome to Spice")},
		},
	})
	if err != nil {
		fmt.Println("resource loader unavailable")
		return
	}
	welcome, err := loader.Resolve(
		"spice://assets/messages/welcome.txt",
	)
	if err != nil {
		fmt.Println("resource unavailable")
		return
	}
	content, err := welcome.ReadAll(context.Background(), 1024)
	if err != nil {
		fmt.Println("resource read failed")
		return
	}
	fmt.Println(string(content))
	// Output: Welcome to Spice
}
