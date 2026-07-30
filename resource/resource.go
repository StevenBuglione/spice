// Package resource provides instance-owned, rooted access to explicitly
// mounted Go filesystems. It performs no network access or package scanning.
package resource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

const (
	scheme            = "spice"
	maxLocationBytes  = 4096
	maxMountNameBytes = 128
	maxReadBytes      = int64(64 << 20)
)

var mountNamePattern = regexp.MustCompile(`^[a-z][a-z0-9.-]*$`)

// Mount binds one stable name to one caller-owned filesystem.
type Mount struct {
	Name string
	FS   fs.FS
}

// Loader resolves canonical spice://<mount>/<path> resource locations.
// Loader is immutable after construction and safe for concurrent use when its
// mounted filesystems are safe for concurrent use.
type Loader struct {
	mounts map[string]fs.FS
	names  []string
}

// NewLoader validates and freezes explicit filesystem mounts.
func NewLoader(mounts ...Mount) (*Loader, error) {
	loader := &Loader{
		mounts: make(map[string]fs.FS, len(mounts)),
		names:  make([]string, 0, len(mounts)),
	}
	for index, mount := range mounts {
		if len(mount.Name) == 0 ||
			len(mount.Name) > maxMountNameBytes ||
			!mountNamePattern.MatchString(mount.Name) {
			return nil, fmt.Errorf(
				"resource mount %d name must be a bounded lowercase identifier",
				index,
			)
		}
		if mount.FS == nil {
			return nil, fmt.Errorf(
				"resource mount %q filesystem is nil",
				mount.Name,
			)
		}
		if _, duplicate := loader.mounts[mount.Name]; duplicate {
			return nil, fmt.Errorf(
				"resource mount %q is declared more than once",
				mount.Name,
			)
		}
		loader.mounts[mount.Name] = mount.FS
		loader.names = append(loader.names, mount.Name)
	}
	slices.Sort(loader.names)
	return loader, nil
}

// Mounts returns mounted names in lexical order.
func (loader *Loader) Mounts() []string {
	if loader == nil {
		return nil
	}
	return slices.Clone(loader.names)
}

// Resolve validates one canonical location without opening it.
func (loader *Loader) Resolve(location string) (Resource, error) {
	if loader == nil {
		return Resource{}, errors.New("resource loader is nil")
	}
	mountName, resourcePath, err := parseLocation(location)
	if err != nil {
		return Resource{}, err
	}
	source, found := loader.mounts[mountName]
	if !found {
		return Resource{}, fmt.Errorf(
			"resolve resource %q: mount %q is not configured",
			location,
			mountName,
		)
	}
	return Resource{
		location: location,
		mount:    mountName,
		path:     resourcePath,
		source:   source,
	}, nil
}

// Resource is one resolved filesystem entry. It retains no open handle.
type Resource struct {
	location string
	mount    string
	path     string
	source   fs.FS
}

// Location returns the canonical resource location.
func (resource Resource) Location() string {
	return resource.location
}

// Mount returns the configured filesystem mount name.
func (resource Resource) Mount() string {
	return resource.mount
}

// Path returns the slash-separated fs.ValidPath within the mount.
func (resource Resource) Path() string {
	return resource.path
}

// Open opens a fresh caller-owned handle after checking cancellation.
func (resource Resource) Open(ctx context.Context) (fs.File, error) {
	if err := resource.validateContext(ctx); err != nil {
		return nil, err
	}
	file, err := resource.source.Open(resource.path)
	if err != nil {
		return nil, fmt.Errorf(
			"open resource %q: %w",
			resource.location,
			err,
		)
	}
	if cause := context.Cause(ctx); cause != nil {
		return nil, errors.Join(
			fmt.Errorf("open resource %q: %w", resource.location, cause),
			file.Close(),
		)
	}
	return file, nil
}

// Stat returns current resource metadata without retaining an open handle.
func (resource Resource) Stat(ctx context.Context) (fs.FileInfo, error) {
	if err := resource.validateContext(ctx); err != nil {
		return nil, err
	}
	info, err := fs.Stat(resource.source, resource.path)
	if err != nil {
		return nil, fmt.Errorf(
			"stat resource %q: %w",
			resource.location,
			err,
		)
	}
	if cause := context.Cause(ctx); cause != nil {
		return nil, fmt.Errorf(
			"stat resource %q: %w",
			resource.location,
			cause,
		)
	}
	return info, nil
}

// Sub returns a filesystem rooted at a directory resource.
func (resource Resource) Sub(ctx context.Context) (fs.FS, error) {
	info, err := resource.Stat(ctx)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(
			"sub resource %q: resource is not a directory",
			resource.location,
		)
	}
	sub, err := fs.Sub(resource.source, resource.path)
	if err != nil {
		return nil, fmt.Errorf(
			"sub resource %q: %w",
			resource.location,
			err,
		)
	}
	if cause := context.Cause(ctx); cause != nil {
		return nil, fmt.Errorf(
			"sub resource %q: %w",
			resource.location,
			cause,
		)
	}
	return sub, nil
}

// ReadAll reads a non-directory resource with an explicit byte limit.
func (resource Resource) ReadAll(
	ctx context.Context,
	limit int64,
) (result []byte, resultErr error) {
	if limit <= 0 || limit > maxReadBytes {
		return nil, fmt.Errorf(
			"read resource %q: limit must be between 1 and %d bytes",
			resource.location,
			maxReadBytes,
		)
	}
	file, err := resource.Open(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		resultErr = errors.Join(resultErr, file.Close())
	}()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf(
			"read resource %q metadata: %w",
			resource.location,
			err,
		)
	}
	if info.IsDir() {
		return nil, fmt.Errorf(
			"read resource %q: resource is a directory",
			resource.location,
		)
	}
	content, err := io.ReadAll(io.LimitReader(
		contextReader{
			cause:  func() error { return context.Cause(ctx) },
			reader: file,
		},
		limit+1,
	))
	if err != nil {
		return nil, fmt.Errorf(
			"read resource %q: %w",
			resource.location,
			err,
		)
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf(
			"read resource %q: content exceeds %d bytes",
			resource.location,
			limit,
		)
	}
	return content, nil
}

func (resource Resource) validateContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf(
			"access resource %q: context is nil",
			resource.location,
		)
	}
	if resource.source == nil ||
		resource.location == "" ||
		resource.mount == "" ||
		!fs.ValidPath(resource.path) {
		return errors.New("resource is not initialized")
	}
	if cause := context.Cause(ctx); cause != nil {
		return fmt.Errorf(
			"access resource %q: %w",
			resource.location,
			cause,
		)
	}
	return nil
}

type contextReader struct {
	cause  func() error
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	if cause := reader.cause(); cause != nil {
		return 0, cause
	}
	return reader.reader.Read(buffer)
}

func parseLocation(location string) (string, string, error) {
	if len(location) == 0 || len(location) > maxLocationBytes {
		return "", "", errors.New(
			"resource location must be non-empty and bounded",
		)
	}
	parsed, err := url.Parse(location)
	if err != nil {
		return "", "", fmt.Errorf(
			"parse resource location: %w",
			err,
		)
	}
	if err := validateParsedLocation(location, parsed); err != nil {
		return "", "", err
	}
	resourcePath := strings.TrimPrefix(parsed.Path, "/")
	if !validResourcePath(resourcePath) {
		return "", "", fmt.Errorf(
			"resource location %q has an invalid path",
			location,
		)
	}
	if parsed.Path != "/"+resourcePath {
		return "", "", fmt.Errorf(
			"resource location %q is not canonical",
			location,
		)
	}
	return parsed.Host, resourcePath, nil
}

func validateParsedLocation(location string, parsed *url.URL) error {
	if parsed.Scheme != scheme ||
		parsed.Host == "" ||
		parsed.User != nil ||
		parsed.RawQuery != "" ||
		parsed.Fragment != "" ||
		parsed.RawPath != "" ||
		parsed.Opaque != "" {
		return fmt.Errorf(
			"resource location %q must use canonical spice://mount/path syntax",
			location,
		)
	}
	if !mountNamePattern.MatchString(parsed.Host) {
		return fmt.Errorf(
			"resource location %q has an invalid mount",
			location,
		)
	}
	return nil
}

func validResourcePath(resourcePath string) bool {
	return resourcePath != "" &&
		fs.ValidPath(resourcePath) &&
		!strings.Contains(resourcePath, `\`)
}
