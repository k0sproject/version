package version

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/k0sproject/version/internal/cache"
	"github.com/k0sproject/version/internal/github"
)

// CacheMaxAge is the maximum duration a cached version list is considered fresh
// before forcing a refresh from GitHub.
const CacheMaxAge = 60 * time.Minute

// ErrCacheMiss is returned when no cached version data is available.
var ErrCacheMiss = errors.New("version: cache miss")

// Collection is a type that implements the sort.Interface interface
// so that versions can be sorted.
type Collection []*Version

func NewCollection(versions ...string) (Collection, error) {
	c := make(Collection, len(versions))
	for i, v := range versions {
		nv, err := NewVersion(v)
		if err != nil {
			return Collection{}, fmt.Errorf("invalid version '%s': %w", v, err)
		}
		c[i] = nv
	}
	return c, nil
}

func (c Collection) Len() int {
	return len(c)
}

func (c Collection) Less(i, j int) bool {
	return c[i].Compare(c[j]) < 0
}

func (c Collection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

// newCollectionFromCache returns the cached versions and the file's modification time.
// It returns ErrCacheMiss when no usable cache exists.
func newCollectionFromCache() (Collection, time.Time, error) {
	path, err := cache.File()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("locate cache: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, time.Time{}, ErrCacheMiss
		}
		return nil, time.Time{}, fmt.Errorf("open cache: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	info, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("stat cache: %w", err)
	}

	collection, readErr := readCollection(f)
	if readErr != nil {
		return nil, time.Time{}, fmt.Errorf("read cache: %w", readErr)
	}
	if len(collection) == 0 {
		return nil, info.ModTime(), ErrCacheMiss
	}

	return collection, info.ModTime(), nil
}

// writeCache persists the collection to the cache file, one version per line.
func (c Collection) writeCache() error {
	path, err := cache.File()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	ordered := slices.Clone(c)
	ordered = slices.DeleteFunc(ordered, func(v *Version) bool {
		return v == nil
	})
	slices.SortFunc(ordered, func(a, b *Version) int {
		return a.Compare(b)
	})
	slices.Reverse(ordered)

	var b strings.Builder
	for _, v := range ordered {
		b.WriteString(v.String())
		b.WriteByte('\n')
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// All returns all known k0s versions using the provided context. It refreshes
// the local cache by querying GitHub for tags newer than the cache
// modification time when the cache is older than CacheMaxAge. The cache is
// skipped if the remote lookup fails and no cached data exists.
func All(ctx context.Context) (Collection, error) {
	result, err := loadAll(ctx, defaultHTTPClient(), false)
	return result.versions, err
}

// Refresh fetches versions from GitHub regardless of cache freshness, updating the cache on success.
func Refresh() (Collection, error) {
	return RefreshContext(context.Background())
}

// RefreshContext fetches versions from GitHub regardless of cache freshness,
// updating the cache on success using the provided context.
func RefreshContext(ctx context.Context) (Collection, error) {
	result, err := loadAll(ctx, defaultHTTPClient(), true)
	return result.versions, err
}

type loadResult struct {
	versions     Collection
	usedFallback bool
}

func loadAll(ctx context.Context, httpClient *http.Client, force bool) (loadResult, error) {
	cached, modTime, cacheErr := newCollectionFromCache()
	if cacheErr != nil && !errors.Is(cacheErr, ErrCacheMiss) {
		return loadResult{}, cacheErr
	}

	known := make(map[string]*Version, len(cached))
	for _, v := range cached {
		if v == nil {
			continue
		}
		known[v.String()] = v
	}

	cacheStale := force || errors.Is(cacheErr, ErrCacheMiss) || modTime.IsZero() || time.Since(modTime) > CacheMaxAge
	if !cacheStale {
		return loadResult{versions: collectionFromMap(known)}, nil
	}

	client := github.NewClient(httpClient)
	tags, err := client.TagsSince(ctx, modTime)
	if err != nil {
		if force || len(known) == 0 {
			return loadResult{}, err
		}
		return loadResult{versions: collectionFromMap(known), usedFallback: true}, nil
	}

	var updated bool
	for _, tag := range tags {
		version, err := NewVersion(tag)
		if err != nil {
			continue
		}
		key := version.String()
		if _, exists := known[key]; exists {
			continue
		}
		known[key] = version
		updated = true
	}

	result := collectionFromMap(known)

	if updated || errors.Is(cacheErr, ErrCacheMiss) || force {
		if err := result.writeCache(); err != nil {
			return loadResult{}, err
		}
	}

	return loadResult{versions: result}, nil
}

func collectionFromMap(m map[string]*Version) Collection {
	if len(m) == 0 {
		return nil
	}
	values := slices.Collect(maps.Values(m))
	values = slices.DeleteFunc(values, func(v *Version) bool {
		return v == nil
	})
	slices.SortFunc(values, func(a, b *Version) int {
		return a.Compare(b)
	})
	return Collection(values)
}

func readCollection(r io.Reader) (Collection, error) {
	var collection Collection
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		v, err := NewVersion(line)
		if err != nil {
			continue
		}
		collection = append(collection, v)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return collection, nil
}
