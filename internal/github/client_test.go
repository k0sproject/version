package github_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalgithub "github.com/k0sproject/version/internal/github"
)

func TestTagsSinceDoesNotFetchCommits(t *testing.T) {
	since := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)

	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/k0sproject/k0s/tags":
			if got, want := r.Header.Get("If-Modified-Since"), since.UTC().Format(http.TimeFormat); got != want {
				t.Fatalf("expected If-Modified-Since header %q, got %q", want, got)
			}

			w.Header().Set("Content-Type", "application/json")
			if _, err := fmt.Fprintf(w, `[
	                {"name":"v1.2.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/a1"}},
	                {"name":"v1.1.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/a2"}}
	            ]`, serverURL, serverURL); err != nil {
				t.Fatalf("write tags response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	client := internalgithub.NewClient(server.Client())

	got, err := client.TagsSince(context.Background(), since)
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

	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/k0sproject/k0s/tags":
			page := r.URL.Query().Get("page")
			w.Header().Set("Content-Type", "application/json")
			switch page {
			case "1":
				w.Header().Set("Link", fmt.Sprintf("<%s/repos/k0sproject/k0s/tags?page=2>; rel=\"next\"", serverURL))
				if _, err := fmt.Fprintf(w, `[
	                    {"name":"v1.4.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/c1"}}
	                ]`, serverURL); err != nil {
					t.Fatalf("write page1 tags: %v", err)
				}
			case "2":
				if _, err := fmt.Fprintf(w, `[
	                    {"name":"v1.3.0","commit":{"url":"%s/repos/k0sproject/k0s/commits/c2"}}
	                ]`, serverURL); err != nil {
					t.Fatalf("write page2 tags: %v", err)
				}
			default:
				if _, err := fmt.Fprint(w, `[]`); err != nil {
					t.Fatalf("write empty page: %v", err)
				}
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	t.Setenv("GITHUB_API_URL", server.URL)
	t.Setenv("GITHUB_TOKEN", "")

	client := internalgithub.NewClient(server.Client())

	got, err := client.TagsSince(context.Background(), since)
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
	got, err := client.TagsSince(context.Background(), since)
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
