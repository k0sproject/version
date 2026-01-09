package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL  = "https://api.github.com"
	repoOwner       = "k0sproject"
	repoName        = "k0s"
	perPage         = 100
	headerAccept    = "application/vnd.github+json"
	headerUserAgent = "github.com/k0sproject/version"
)

// Client wraps GitHub REST usage tailored for listing tags.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a GitHub client. If httpClient is nil a default
// client with a 10s timeout is used. The base URL can be overridden via
// the GITHUB_API_URL environment variable (useful for tests or GHES).
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	base := os.Getenv("GITHUB_API_URL")
	if base == "" {
		base = defaultBaseURL
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(base, "/"),
		token:      strings.TrimSpace(os.Getenv("GITHUB_TOKEN")),
	}
}

// TagsSince returns tag names that GitHub reports as updated since the provided time.
// When since is zero, all tags are returned (subject to pagination of the tags
// endpoint itself).
func (c *Client) TagsSince(ctx context.Context, since time.Time) ([]string, error) {
	if c == nil {
		return nil, errors.New("github client is nil")
	}

	if c.httpClient == nil {
		return nil, errors.New("http client is nil")
	}

	sinceHeader := ""
	if !since.IsZero() {
		sinceHeader = since.UTC().Format(http.TimeFormat)
	}

	var tags []string

	for page := 1; ; page++ {
		tagsURL := fmt.Sprintf("%s/%s", strings.TrimRight(c.baseURL, "/"), path.Join("repos", repoOwner, repoName, "tags"))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, tagsURL, nil)
		if err != nil {
			return nil, err
		}

		q := req.URL.Query()
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))
		req.URL.RawQuery = q.Encode()

		req.Header.Set("Accept", headerAccept)
		req.Header.Set("User-Agent", headerUserAgent)
		if sinceHeader != "" {
			req.Header.Set("If-Modified-Since", sinceHeader)
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode == http.StatusNotModified {
			return tags, nil
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("github tags request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var payload []tagResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return nil, fmt.Errorf("decode tags payload: %w", err)
		}

		if len(payload) == 0 {
			break
		}

		for _, tag := range payload {
			tags = append(tags, tag.Name)
		}

		if !hasNextPage(resp.Header.Get("Link")) {
			break
		}
	}

	return tags, nil
}

func hasNextPage(linkHeader string) bool {
	for _, part := range strings.Split(linkHeader, ",") {
		section := strings.TrimSpace(part)
		if section == "" {
			continue
		}
		if strings.Contains(section, "rel=\"next\"") {
			return true
		}
	}
	return false
}

type tagResponse struct {
	Name string `json:"name"`
}
