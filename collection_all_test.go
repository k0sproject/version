package version

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/k0sproject/version/internal/cache"
)

func TestAllFetchesAndCaches(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/k0sproject/k0s/tags":
			w.Header().Set("Content-Type", "application/json")
			if _, err := fmt.Fprintf(w, `[
                {"name":"v1.25.0+k0s.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/c1"}},
                {"name":"v1.24.3+k0s.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/c2"}}
	            ]`, serverURL, serverURL); err != nil {
				t.Fatalf("write tags response: %v", err)
			}
		case "/repos/k0sproject/k0s/commits/c1":
			w.Header().Set("Content-Type", "application/json")
			if _, err := fmt.Fprint(w, `{"commit":{"committer":{"date":"2024-03-10T00:00:00Z"}}}`); err != nil {
				t.Fatalf("write commit response: %v", err)
			}
		case "/repos/k0sproject/k0s/commits/c2":
			w.Header().Set("Content-Type", "application/json")
			if _, err := fmt.Fprint(w, `{"commit":{"committer":{"date":"2024-02-01T00:00:00Z"}}}`); err != nil {
				t.Fatalf("write commit response: %v", err)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	ctx := context.Background()
	versions, err := All(ctx)
	if err != nil {
		t.Fatalf("All() returned error: %v", err)
	}

	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}

	if versions[0].String() != "v1.24.3+k0s.0" {
		t.Fatalf("unexpected first version %q", versions[0])
	}
	if versions[1].String() != "v1.25.0+k0s.0" {
		t.Fatalf("unexpected second version %q", versions[1])
	}

	cachePath, err := cache.File()
	if err != nil {
		t.Fatalf("cache.File() error: %v", err)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("reading cache: %v", err)
	}

	want := "v1.25.0+k0s.0\nv1.24.3+k0s.0\n"
	if string(data) != want {
		t.Fatalf("cache contents = %q, want %q", string(data), want)
	}
}

func TestAllFallsBackToCacheOnError(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	seed, err := NewCollection("v1.24.3+k0s.0")
	if err != nil {
		t.Fatalf("seeding collection: %v", err)
	}
	if err := seed.writeCache(); err != nil {
		t.Fatalf("priming cache: %v", err)
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer failServer.Close()

	t.Setenv("GITHUB_API_URL", failServer.URL)
	t.Setenv("GITHUB_TOKEN", "")

	ctx := context.Background()
	versions, err := All(ctx)
	if err != nil {
		t.Fatalf("All() returned error: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("expected 1 version from cache, got %d", len(versions))
	}
	if versions[0].String() != "v1.24.3+k0s.0" {
		t.Fatalf("unexpected cached version %q", versions[0])
	}
}

func TestAllReturnsCachedWhenStaleAndRemoteFails(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	seed, err := NewCollection("v1.24.3+k0s.0")
	if err != nil {
		t.Fatalf("seeding collection: %v", err)
	}
	if err := seed.writeCache(); err != nil {
		t.Fatalf("priming cache: %v", err)
	}

	cachePath, err := cache.File()
	if err != nil {
		t.Fatalf("cache.File() error: %v", err)
	}

	older := time.Now().Add(-(CacheMaxAge + time.Minute))
	if err := os.Chtimes(cachePath, older, older); err != nil {
		t.Fatalf("setting cache time: %v", err)
	}

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer failServer.Close()

	t.Setenv("GITHUB_API_URL", failServer.URL)
	t.Setenv("GITHUB_TOKEN", "")

	ctx := context.Background()
	versions, err := All(ctx)
	if err != nil {
		t.Fatalf("All() returned error: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("expected 1 version from cache, got %d", len(versions))
	}
	if versions[0].String() != "v1.24.3+k0s.0" {
		t.Fatalf("unexpected cached version %q", versions[0])
	}
}

func TestRefreshFailsWhenRemoteFails(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	seed, err := NewCollection("v1.24.3+k0s.0")
	if err != nil {
		t.Fatalf("seeding collection: %v", err)
	}
	if err := seed.writeCache(); err != nil {
		t.Fatalf("priming cache: %v", err)
	}

	failure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer failure.Close()

	t.Setenv("GITHUB_API_URL", failure.URL)
	t.Setenv("GITHUB_TOKEN", "")

	if _, err := Refresh(); err == nil {
		t.Fatal("expected error when refresh fails")
	}
}

func TestAllReturnsErrorWhenNoCacheAndRemoteFails(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer failServer.Close()

	t.Setenv("GITHUB_API_URL", failServer.URL)
	t.Setenv("GITHUB_TOKEN", "")

	ctx := context.Background()
	if _, err := All(ctx); err == nil {
		t.Fatal("expected error when remote fails without cache")
	}
}

func TestAllSendsIfModifiedSince(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	seed, err := NewCollection("v1.24.3+k0s.0")
	if err != nil {
		t.Fatalf("seeding collection: %v", err)
	}
	if err := seed.writeCache(); err != nil {
		t.Fatalf("priming cache: %v", err)
	}

	cachePath, err := cache.File()
	if err != nil {
		t.Fatalf("cache.File() error: %v", err)
	}

	older := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cachePath, older, older); err != nil {
		t.Fatalf("setting cache time: %v", err)
	}

	var received string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/k0sproject/k0s/tags" {
			received = r.Header.Get("If-Modified-Since")
			w.WriteHeader(http.StatusNotModified)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	ctx := context.Background()
	versions, err := All(ctx)
	if err != nil {
		t.Fatalf("All() returned error: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected cached version, got %d", len(versions))
	}
	if received == "" {
		t.Fatal("expected If-Modified-Since header to be set")
	}
}
