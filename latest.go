package version

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// LatestByPrerelease returns the latest released k0s version. When allowpre is
// true prereleases are also accepted.
func LatestByPrerelease(allowpre bool) (*Version, error) {
	return LatestByPrereleaseContext(context.Background(), allowpre)
}

// LatestByPrereleaseContext returns the latest released k0s version using the
// provided context. When allowpre is true prereleases are also accepted.
func LatestByPrereleaseContext(ctx context.Context, allowpre bool) (*Version, error) {
	client := defaultHTTPClient()
	result, err := loadAll(ctx, client, false)
	versions := result.versions

	var candidate *Version
	if err == nil {
		candidate = selectLatest(versions, allowpre)
		if candidate != nil && !result.usedFallback {
			return candidate, nil
		}
	}

	fallback, fallbackErr := fetchLatestFromDocs(ctx, docsHTTPClient(), allowpre)
	if fallbackErr == nil {
		if candidate == nil {
			return fallback, nil
		}
		if fallback.GreaterThan(candidate) {
			return fallback, nil
		}
		return candidate, nil
	}

	if candidate != nil {
		return candidate, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	return nil, fallbackErr
}

// LatestStable returns the semantically sorted latest non-prerelease version
// from the cached collection.
func LatestStable() (*Version, error) {
	return LatestByPrerelease(false)
}

// LatestStableContext returns the semantically sorted latest non-prerelease
// version from the cached collection using the provided context.
func LatestStableContext(ctx context.Context) (*Version, error) {
	return LatestByPrereleaseContext(ctx, false)
}

// Latest returns the semantically sorted latest version even if it is a
// prerelease from the cached collection.
func Latest() (*Version, error) {
	return LatestByPrerelease(true)
}

// LatestContext returns the semantically sorted latest version even if it is a
// prerelease from the cached collection using the provided context.
func LatestContext(ctx context.Context) (*Version, error) {
	return LatestByPrereleaseContext(ctx, true)
}

func selectLatest(collection Collection, allowpre bool) *Version {
	for i := len(collection) - 1; i >= 0; i-- {
		v := collection[i]
		if v == nil {
			continue
		}
		if !allowpre && v.IsPrerelease() {
			continue
		}
		return v
	}
	return nil
}

func fetchLatestFromDocs(ctx context.Context, client *http.Client, allowpre bool) (*Version, error) {
	path := "stable.txt"
	if allowpre {
		path = "latest.txt"
	}

	base := strings.TrimSpace(os.Getenv("K0S_VERSION_DOCS_BASE_URL"))
	if base == "" {
		base = "https://docs.k0sproject.io"
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("parse docs base url %q: %w", base, err)
	}
	baseURL.Path = path

	text, err := httpGet(ctx, client, baseURL.String())
	if err != nil {
		return nil, err
	}

	return NewVersion(text)
}

func httpGet(ctx context.Context, client *http.Client, u string) (string, error) {
	if client == nil {
		client = defaultHTTPClient()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("create request for %s: %w", u, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request to %s failed: %w", u, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.Body == nil {
		return "", fmt.Errorf("http request to %s failed: nil body", u)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http request to %s failed: backend returned %d", u, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("http request to %s failed: %w when reading body", u, err)
	}

	return strings.TrimSpace(string(body)), nil
}
