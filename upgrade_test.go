package version

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mustVersion(t *testing.T, input string) *Version {
	t.Helper()
	v, err := NewVersion(input)
	if err != nil {
		t.Fatalf("failed to parse version %q: %v", input, err)
	}
	return v
}

func setupTagServer(t *testing.T) *httptest.Server {
	t.Helper()

	tags := []struct {
		name   string
		commit string
		date   string
	}{
		{"v1.26.1+k0s.0", "c1", "2024-03-10T00:00:00Z"},
		{"v1.26.0+k0s.0", "c2", "2024-03-05T00:00:00Z"},
		{"v1.26.0-rc.1+k0s.0", "c3", "2024-02-25T00:00:00Z"},
		{"v1.25.1+k0s.0", "c4", "2024-02-10T00:00:00Z"},
		{"v1.25.0+k0s.0", "c5", "2024-01-31T00:00:00Z"},
		{"v1.24.3+k0s.0", "c6", "2024-01-15T00:00:00Z"},
		{"v1.24.1+k0s.0", "c7", "2023-12-20T00:00:00Z"},
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/k0sproject/k0s/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, "[")
			for i, tag := range tags {
				if i > 0 {
					_, _ = fmt.Fprint(w, ",")
				}
				_, _ = fmt.Fprintf(w, "{\"name\":\"%s\",\"commit\":{\"url\":\"%s/repos/k0sproject/k0s/commits/%s\"}}", tag.name, server.URL, tag.commit)
			}
			_, _ = fmt.Fprint(w, "]")
		default:
			for _, tag := range tags {
				if r.URL.Path == fmt.Sprintf("/repos/k0sproject/k0s/commits/%s", tag.commit) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, "{\"commit\":{\"committer\":{\"date\":\"%s\"}}}", tag.date)
					return
				}
			}
			http.NotFound(w, r)
		}
	}))

	t.Cleanup(server.Close)
	return server
}

func TestUpgradePathToStableTarget(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	server := setupTagServer(t)
	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	current := mustVersion(t, "v1.24.1+k0s.0")
	target := mustVersion(t, "v1.26.1+k0s.0")

	path, err := current.UpgradePath(target)
	if err != nil {
		t.Fatalf("UpgradePath returned error: %v", err)
	}

	got := versionsToStrings(path)
	want := []string{"v1.24.3+k0s.0", "v1.25.1+k0s.0", "v1.26.1+k0s.0"}
	if len(got) != len(want) {
		t.Fatalf("expected %d steps, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected step %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestUpgradePathToPrereleaseTarget(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	server := setupTagServer(t)
	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	current := mustVersion(t, "v1.24.1+k0s.0")
	target := mustVersion(t, "v1.26.0-rc.1+k0s.0")

	path, err := current.UpgradePath(target)
	if err != nil {
		t.Fatalf("UpgradePath returned error: %v", err)
	}

	got := versionsToStrings(path)
	want := []string{"v1.24.3+k0s.0", "v1.25.1+k0s.0", "v1.26.0-rc.1+k0s.0"}
	if len(got) != len(want) {
		t.Fatalf("expected %d steps, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected step %d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestUpgradePathRejectsDowngrade(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	server := setupTagServer(t)
	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	current := mustVersion(t, "v1.25.0+k0s.0")
	target := mustVersion(t, "v1.24.3+k0s.0")

	if _, err := current.UpgradePath(target); err == nil {
		t.Fatal("expected downgrade error")
	}
}

func versionsToStrings(collection Collection) []string {
	out := make([]string, 0, len(collection))
	for _, v := range collection {
		if v == nil {
			continue
		}
		out = append(out, v.String())
	}
	return out
}
