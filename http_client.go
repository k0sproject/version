package version

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultGitHubAPIBaseURL = "https://api.github.com"
	defaultDocsBaseURL      = "https://docs.k0sproject.io"
)

var (
	// GitHubAPIURL controls the base URL used when querying the GitHub API.
	// When empty, the value is resolved from the GITHUB_API_URL environment
	// variable and ultimately falls back to the public api.github.com endpoint.
	GitHubAPIURL = strings.TrimSpace(os.Getenv("GITHUB_API_URL"))

	// LatestURL overrides the URL used to fetch the latest version reference
	// that may include prereleases. When empty, the value is derived from
	// K0S_VERSION_DOCS_BASE_URL or the public docs endpoint.
	LatestURL string

	// StableURL overrides the URL used to fetch the stable version reference.
	// When empty, the value is derived from K0S_VERSION_DOCS_BASE_URL or the
	// public docs endpoint.
	StableURL string

	// HTTPTimeout bounds the overall duration of HTTP requests issued by the
	// package level helpers. A non-positive value disables the client timeout.
	HTTPTimeout = 10 * time.Second

	// HTTPConnectTimeout bounds how long a dial attempt is allowed to take.
	// A non-positive value allows connections to use the net/http defaults.
	HTTPConnectTimeout = 5 * time.Second

	// HTTPReadTimeout bounds how long the client waits for response headers.
	// A non-positive value keeps the net/http defaults.
	HTTPReadTimeout = 5 * time.Second

	defaultClientMu sync.RWMutex
	defaultClient   *http.Client

	docsClientMu sync.RWMutex
	docsClient   *http.Client
)

type httpClientTarget uint8

const (
	httpClientTargetGitHub httpClientTarget = iota
	httpClientTargetDocs
)

type httpClientContextKey struct {
	target httpClientTarget
}

type httpTimeoutKey struct {
	target httpClientTarget
}

var (
	httpClientKeyGitHub = httpClientContextKey{target: httpClientTargetGitHub}
	httpClientKeyDocs   = httpClientContextKey{target: httpClientTargetDocs}

	httpTimeoutKeyGitHub = httpTimeoutKey{target: httpClientTargetGitHub}
	httpTimeoutKeyDocs   = httpTimeoutKey{target: httpClientTargetDocs}
)

// ContextWithHTTPClient returns a context configured to use client for GitHub lookups.
func ContextWithHTTPClient(ctx context.Context, client *http.Client) context.Context {
	return withHTTPClient(ctx, httpClientKeyGitHub, client)
}

// ContextWithDocsHTTPClient returns a context configured to use client for documentation lookups.
func ContextWithDocsHTTPClient(ctx context.Context, client *http.Client) context.Context {
	return withHTTPClient(ctx, httpClientKeyDocs, client)
}

// ContextWithHTTPTimeout returns a context configured with a per-request timeout for GitHub lookups.
// A non-positive timeout disables the default and keeps the existing deadline behavior.
func ContextWithHTTPTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return withHTTPTimeout(ctx, httpTimeoutKeyGitHub, timeout)
}

// ContextWithDocsHTTPTimeout returns a context configured with a per-request timeout for documentation lookups.
// A non-positive timeout disables the default and keeps the existing deadline behavior.
func ContextWithDocsHTTPTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return withHTTPTimeout(ctx, httpTimeoutKeyDocs, timeout)
}

func withHTTPClient(ctx context.Context, key httpClientContextKey, client *http.Client) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		return ctx
	}
	return context.WithValue(ctx, key, client)
}

func httpClientFromContext(ctx context.Context, key httpClientContextKey, fallback func() *http.Client) *http.Client {
	if ctx != nil {
		if client, ok := ctx.Value(key).(*http.Client); ok && client != nil {
			return client
		}
	}
	return fallback()
}

func withHTTPTimeout(ctx context.Context, key httpTimeoutKey, timeout time.Duration) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, key, timeout)
}

func timeoutFromContext(ctx context.Context, key httpTimeoutKey) (time.Duration, bool) {
	if ctx == nil {
		return 0, false
	}
	val, ok := ctx.Value(key).(time.Duration)
	return val, ok
}

func withTargetTimeout(ctx context.Context, key httpTimeoutKey) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, nil
	}

	if timeout, ok := timeoutFromContext(ctx, key); ok {
		if timeout <= 0 {
			return ctx, nil
		}
		return context.WithTimeout(ctx, timeout)
	}

	if HTTPTimeout > 0 {
		return context.WithTimeout(ctx, HTTPTimeout)
	}
	return ctx, nil
}

func defaultHTTPClient() *http.Client {
	defaultClientMu.RLock()
	client := defaultClient
	defaultClientMu.RUnlock()
	if client != nil {
		return client
	}

	defaultClientMu.Lock()
	defer defaultClientMu.Unlock()
	if defaultClient == nil {
		defaultClient = newConfiguredHTTPClient()
	}
	return defaultClient
}

func docsHTTPClient() *http.Client {
	docsClientMu.RLock()
	client := docsClient
	docsClientMu.RUnlock()
	if client != nil {
		return client
	}

	docsClientMu.Lock()
	defer docsClientMu.Unlock()
	if docsClient == nil {
		docsClient = newConfiguredHTTPClient()
	}
	return docsClient
}

// SetDefaultHTTPClient overrides the shared HTTP client used for GitHub lookups by helper functions without contexts.
func SetDefaultHTTPClient(client *http.Client) {
	defaultClientMu.Lock()
	defaultClient = client
	defaultClientMu.Unlock()
}

// SetDocsHTTPClient overrides the shared HTTP client used for documentation fallbacks.
func SetDocsHTTPClient(client *http.Client) {
	docsClientMu.Lock()
	docsClient = client
	docsClientMu.Unlock()
}

func newConfiguredHTTPClient() *http.Client {
	baseTransport, _ := http.DefaultTransport.(*http.Transport)
	var transport *http.Transport
	if baseTransport != nil {
		transport = baseTransport.Clone()
	} else {
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
	}

	if HTTPConnectTimeout > 0 {
		dialer := &net.Dialer{
			Timeout:   HTTPConnectTimeout,
			KeepAlive: 30 * time.Second,
		}
		transport.DialContext = dialer.DialContext
		transport.TLSHandshakeTimeout = HTTPConnectTimeout
	}

	if HTTPReadTimeout > 0 {
		transport.ResponseHeaderTimeout = HTTPReadTimeout
	}

	client := &http.Client{Transport: transport}
	if HTTPTimeout > 0 {
		client.Timeout = HTTPTimeout
	}
	return client
}

func githubAPIURL() string {
	if base := strings.TrimSpace(GitHubAPIURL); base != "" {
		return strings.TrimRight(base, "/")
	}
	if base := strings.TrimSpace(os.Getenv("GITHUB_API_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}
	return defaultGitHubAPIBaseURL
}

func docsURL(includePrerelease bool) string {
	if includePrerelease {
		if u := strings.TrimSpace(LatestURL); u != "" {
			return u
		}
		return docsBaseURL() + "/latest.txt"
	}
	if u := strings.TrimSpace(StableURL); u != "" {
		return u
	}
	return docsBaseURL() + "/stable.txt"
}

func docsBaseURL() string {
	if base := strings.TrimSpace(os.Getenv("K0S_VERSION_DOCS_BASE_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}
	return defaultDocsBaseURL
}
