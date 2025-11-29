package version_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/k0sproject/version"
)

func TestLatestByPrerelease(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	repoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/k0sproject/k0s/tags" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `[
    {"name":"v1.25.0+k0s.0"},
    {"name":"v1.24.3+k0s.0"}
    ]`)
			return
		}
		http.NotFound(w, r)
	}))
	defer repoServer.Close()

	docsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stable.txt":
			_, _ = fmt.Fprint(w, "v1.25.1+k0s.0")
		case "/latest.txt":
			_, _ = fmt.Fprint(w, "v1.26.0+k0s.0-rc.1")
		default:
			http.NotFound(w, r)
		}
	}))
	defer docsServer.Close()

	t.Setenv("GITHUB_API_URL", repoServer.URL)
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("K0S_VERSION_DOCS_BASE_URL", docsServer.URL)

	stable, err := version.LatestByPrerelease(false)
	NoError(t, err)
	Equal(t, "v1.25.0+k0s.0", stable.String())

	cachePath := filepath.Join(os.Getenv("XDG_CACHE_HOME"), "k0s_version", "known_versions.txt")
	stale := time.Now().Add(-(version.CacheMaxAge + time.Minute))
	if err := os.Chtimes(cachePath, stale, stale); err != nil {
		t.Fatalf("setting cache stale: %v", err)
	}

	repoServer.Close()

	stableFallback, err := version.LatestByPrerelease(false)
	NoError(t, err)
	Equal(t, "v1.25.1+k0s.0", stableFallback.String())

	latest, err := version.LatestByPrerelease(true)
	NoError(t, err)
	Equal(t, "v1.26.0+k0s.0-rc.1", latest.String())

	ctxLatest, err := version.LatestByPrereleaseContext(context.Background(), true)
	NoError(t, err)
	Equal(t, latest.String(), ctxLatest.String())

	defaultLatest, err := version.Latest()
	NoError(t, err)
	Equal(t, latest.String(), defaultLatest.String())

	ctxDefaultLatest, err := version.LatestContext(context.Background())
	NoError(t, err)
	Equal(t, latest.String(), ctxDefaultLatest.String())
}
