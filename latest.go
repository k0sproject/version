package version

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Latest returns the latest released k0s version, including prereleases.
func Latest() (*Version, error) {
	return LatestByPrerelease(true)
}

// LatestContext returns the latest released k0s version using the provided
// context, including prereleases.
func LatestContext(ctx context.Context) (*Version, error) {
	return LatestByPrereleaseContext(ctx, true)
}

// LatestByPrerelease returns the latest released k0s version. When allowpre is
// true prereleases are also accepted.
func LatestByPrerelease(allowpre bool) (*Version, error) {
	return LatestByPrereleaseContext(context.Background(), allowpre)
}

// LatestByPrereleaseContext returns the latest released k0s version using the
// provided context. When allowpre is true prereleases are also accepted.
func LatestByPrereleaseContext(ctx context.Context, allowpre bool) (*Version, error) {
	client := httpClientFromContext(ctx, httpClientKeyGitHub, defaultHTTPClient)
	ctxWithTimeout, cancel := withTargetTimeout(ctx, httpTimeoutKeyGitHub)
	if cancel != nil {
		defer cancel()
	}
	result, err := loadAll(ctxWithTimeout, client, false)
	versions := result.versions

	var candidate *Version
	if err == nil {
		candidate = selectLatest(versions, allowpre)
		if candidate != nil && !result.usedFallback {
			return candidate, nil
		}
	}

	docsClient := httpClientFromContext(ctx, httpClientKeyDocs, docsHTTPClient)
	docsCtx, docsCancel := withTargetTimeout(ctx, httpTimeoutKeyDocs)
	if docsCancel != nil {
		defer docsCancel()
	}
	fallback, fallbackErr := fetchLatestFromDocs(docsCtx, docsClient, allowpre)
	if fallbackErr == nil {
		if candidate == nil || fallback.GreaterThan(candidate) {
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
	target := docsURL(allowpre)

	text, err := httpGet(ctx, client, target)
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
