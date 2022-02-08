package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

var TimeOut = time.Second * 10
var Repo = "k0sproject/k0s"

// Release is a k0s release
type Release struct {
	URL        string  `json:"html_url"`
	TagName    string  `json:"tag_name"`
	PreRelease bool    `json:"prerelease"`
	Assets     []Asset `json:"assets"`
}

// Asset describes a release asset
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// GreaterThan returns true if the version of the Release is greater than the supplied version
func (r *Release) GreaterThan(b fmt.Stringer) bool {
	a, err := NewVersion(r.TagName)
	if err != nil {
		return false
	}
	other, err := NewVersion(b.String())
	if err != nil {
		return false
	}
	return a.GreaterThan(other)
}

func (r *Release) Version() (*Version, error) {
	return NewVersion(r.TagName)
}

func (r *Release) String() string {
	return strings.TrimPrefix(r.TagName, "v")
}

func LatestReleaseByPrerelease(allowpre bool) (Release, error) {
	var releases []Release
	if err := unmarshalURLBody(fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20&page=1", Repo), &releases); err != nil {
		return Release{}, err
	}

	var c Collection
	for _, v := range releases {
		if v.PreRelease && !allowpre {
			continue
		}
		if version, err := NewVersion(v.TagName); err == nil {
			c = append(c, version)
		}
	}

	if len(c) == 0 {
		return Release{}, fmt.Errorf("failed to get the latest version information")
	}

	sort.Sort(c)
	latest := c[len(c)-1].String()

	for _, v := range releases {
		if strings.TrimPrefix(v.TagName, "v") == latest {
			return v, nil
		}
	}

	return Release{}, fmt.Errorf("failed to get the latest version information")
}

// LatestStableRelease returns the semantically sorted latest non-prerelease version from the online repository
func LatestStableRelease() (Release, error) {
	return LatestReleaseByPrerelease(false)
}

// LatestStableRelease returns the semantically sorted latest version even if it is a prerelease from the online repository
func LatestRelease() (Release, error) {
	return LatestReleaseByPrerelease(true)
}

func unmarshalURLBody(url string, o interface{}) error {
	client := &http.Client{
		Timeout: TimeOut,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	if resp.Body == nil {
		return fmt.Errorf("nil body")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("backend returned http %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := resp.Body.Close(); err != nil {
		return err
	}

	return json.Unmarshal(body, o)
}
