package cache_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/k0sproject/version/internal/cache"
)

func TestFileHonorsXDGCacheHome(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "xdg"))

	got, err := cache.File()
	if err != nil {
		t.Fatalf("File() returned error: %v", err)
	}

	want := filepath.Join(tmp, "xdg", "k0s_version", "known_versions.txt")
	if got != want {
		t.Fatalf("File() = %q, want %q", got, want)
	}
}

func TestFileProvidesPlatformDefault(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	if runtime.GOOS == "windows" {
		t.Setenv("LOCALAPPDATA", t.TempDir())
	}

	got, err := cache.File()
	if err != nil {
		t.Fatalf("File() returned error: %v", err)
	}

	suffix := filepath.Join("k0s_version", "known_versions.txt")
	if !strings.HasSuffix(got, suffix) {
		t.Fatalf("File() path %q does not end with %q", got, suffix)
	}
}
