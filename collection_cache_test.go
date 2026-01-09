package version

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/k0sproject/version/internal/cache"
)

func TestCollectionWriteCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	c, err := NewCollection("v1.0.0+k0s.1", "v1.0.1+k0s.0")
	if err != nil {
		t.Fatalf("NewCollection() error = %v", err)
	}

	if err := c.writeCache(); err != nil {
		t.Fatalf("writeCache() error = %v", err)
	}

	path, err := cache.File()
	if err != nil {
		t.Fatalf("cache.File() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	want := "v1.0.1+k0s.0\nv1.0.0+k0s.1\n"
	if string(data) != want {
		t.Fatalf("cache contents = %q, want %q", string(data), want)
	}
}

func TestNewCollectionFromCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	path, err := cache.File()
	if err != nil {
		t.Fatalf("cache.File() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	contents := "v1.0.0+k0s.1\ninvalid\n#comment\n\n"
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	collection, modTime, err := newCollectionFromCache()
	if err != nil {
		t.Fatalf("newCollectionFromCache() error = %v", err)
	}
	if modTime.IsZero() {
		t.Fatal("expected modTime to be set")
	}

	if len(collection) != 1 {
		t.Fatalf("expected 1 version, got %d", len(collection))
	}

	if got := collection[0].String(); got != "v1.0.0+k0s.1" {
		t.Fatalf("unexpected version %q", got)
	}

	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if !modTime.Equal(stat.ModTime()) {
		t.Fatalf("modTime %v should match file mod time %v", modTime, stat.ModTime())
	}
}

func TestNewCollectionFromCacheMiss(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	_, _, err := newCollectionFromCache()
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss, got %v", err)
	}
}
