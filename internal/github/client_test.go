package github_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalgithub "github.com/k0sproject/version/internal/github"
)

func TestTagsSinceDoesNotFetchCommits(t *testing.T) {
	since := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/k0sproject/k0s/tags":
			if got, want := r.Header.Get("If-Modified-Since"), since.UTC().Format(http.TimeFormat); got != want {
				t.Fatalf("expected If-Modified-Since header %q, got %q", want, got)
			}

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `[
                {"name":"v1.2.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/a1"}},
                {"name":"v1.1.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/a2"}}
            ]`, server.URL, server.URL)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	client := internalgithub.NewClient(server.Client())

	got, err := client.TagsSince(since)
	if err != nil {
		t.Fatalf("TagsSince returned error: %v", err)
	}

	want := []string{"v1.2.0", "v1.1.0"}
	if len(got) != len(want) {
		t.Fatalf("expected %d tags, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected tag[%d]: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestTagsSinceHandlesPagination(t *testing.T) {
	since := time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC)

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/k0sproject/k0s/tags":
			page := r.URL.Query().Get("page")
			w.Header().Set("Content-Type", "application/json")
			switch page {
			case "1":
				w.Header().Set("Link", fmt.Sprintf("<%s/repos/k0sproject/k0s/tags?page=2>; rel=\"next\"", server.URL))
				fmt.Fprintf(w, `[
                    {"name":"v1.4.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/c1"}}
                ]`, server.URL)
			case "2":
				fmt.Fprintf(w, `[
                    {"name":"v1.3.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/c2"}}
                ]`, server.URL)
			default:
				fmt.Fprint(w, `[]`)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	client := internalgithub.NewClient(server.Client())

	got, err := client.TagsSince(since)
	if err != nil {
		t.Fatalf("TagsSince returned error: %v", err)
	}

	want := []string{"v1.4.0", "v1.3.0"}
	if len(got) != len(want) {
		t.Fatalf("expected %d tags, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected tag[%d]: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestTagsSinceNotModified(t *testing.T) {
	var seenHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeader = r.Header.Get("If-Modified-Since")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	client := internalgithub.NewClient(server.Client())

	since := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	got, err := client.TagsSince(since)
	if err != nil {
		t.Fatalf("TagsSince returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no tags on 304, got %v", got)
	}

	if seenHeader == "" {
		t.Fatal("expected If-Modified-Since header to be sent")
	}
}
