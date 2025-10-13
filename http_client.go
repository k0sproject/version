package version

import (
	"net/http"

	"github.com/k0sproject/version/internal/httpclient"
)

// Timeout controls the default HTTP client timeout for remote lookups.
var Timeout = httpclient.DefaultTimeout

func defaultHTTPClient() *http.Client {
	return httpclient.New(Timeout)
}

func docsHTTPClient() *http.Client {
	return httpclient.New(Timeout)
}
