package httpclient

import (
	"net/http"
	"time"
)

const DefaultTimeout = 10 * time.Second

// New returns a fresh HTTP client configured with the provided timeout. Callers
// should supply the desired timeout while relying on the standard library to
// manage connection pooling for the lifetime of the client.
func New(timeout time.Duration) *http.Client {
	timeout = normalizeTimeout(timeout)
	return &http.Client{Timeout: timeout}
}

func normalizeTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultTimeout
	}
	return timeout
}
