package version

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	goversion "github.com/hashicorp/go-version"
)

var BaseUrl = "https://github.com/k0sproject/k0s/"

// Version is a k0s version
type Version struct {
	goversion.Version
}

func pair(a, b *Version) Collection {
	return Collection{a, b}
}

// String returns a v-prefixed string representation of the k0s version
func (v *Version) String() string {
	if v == nil {
		return ""
	}
	plain := strings.TrimPrefix(v.Version.String(), "v")
	if plain == "" {
		return ""
	}
	return fmt.Sprintf("v%s", plain)
}

func (v *Version) urlString() string {
	return strings.ReplaceAll(v.String(), "+", "%2B")
}

// URL returns an URL to the release information page for the k0s version
func (v *Version) URL() string {
	return BaseUrl + filepath.Join("releases", "tag", v.urlString())
}

func (v *Version) assetBaseURL() string {
	return BaseUrl + filepath.Join("releases", "download", v.urlString()) + "/"
}

// DownloadURL returns the k0s binary download URL for the k0s version
func (v *Version) DownloadURL(os, arch string) string {
	var ext string
	if strings.HasPrefix(strings.ToLower(os), "win") {
		ext = ".exe"
	}
	return v.assetBaseURL() + fmt.Sprintf("k0s-%s-%s%s", v.String(), arch, ext)
}

// AirgapDownloadURL returns the k0s airgap bundle download URL for the k0s version
func (v *Version) AirgapDownloadURL(arch string) string {
	return v.assetBaseURL() + fmt.Sprintf("k0s-airgap-bundle-%s-%s", v.String(), arch)
}

// DocsURL returns the documentation URL for the k0s version
func (v *Version) DocsURL() string {
	return fmt.Sprintf("https://docs.k0sproject.io/%s/", v.String())
}

// Equal returns true if the version is equal to the supplied version
func (v *Version) Equal(b *Version) bool {
	return v.String() == b.String()
}

// GreaterThan returns true if the version is greater than the supplied version
func (v *Version) GreaterThan(b *Version) bool {
	if v.String() == b.String() {
		return false
	}
	p := pair(v, b)
	sort.Sort(p)
	return v.String() == p[1].String()
}

// LessThan returns true if the version is lower than the supplied version
func (v *Version) LessThan(b *Version) bool {
	if v.String() == b.String() {
		return false
	}
	return !v.GreaterThan(b)
}

// GreaterThanOrEqual returns true if the version is greater than the supplied version or equal
func (v *Version) GreaterThanOrEqual(b *Version) bool {
	return v.Equal(b) || v.GreaterThan(b)
}

// LessThanOrEqual returns true if the version is lower than the supplied version or equal
func (v *Version) LessThanOrEqual(b *Version) bool {
	return v.Equal(b) || v.LessThan(b)
}

// Compare compares two versions and returns one of the integers: -1, 0 or 1 (less than, equal, greater than)
func (v *Version) Compare(b *Version) int {
	c := v.Version.Compare(&b.Version)
	if c != 0 {
		return c
	}

	vA := v.String()

	// go to plain string comparison
	s := []string{vA, b.String()}
	sort.Strings(s)

	if vA == s[0] {
		return -1
	}

	return 1
}

// MarshalJSON implements the json.Marshaler interface.
func (v *Version) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	return []byte(fmt.Sprintf("\"%s\"", v.String())), nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (v *Version) MarshalYAML() (interface{}, error) {
	if v == nil {
		return nil, nil
	}

	return v.String(), nil
}

func (v *Version) unmarshal(f func(interface{}) error) error {
	var s string
	if err := f(&s); err != nil {
		return fmt.Errorf("failed to decode input: %w", err)
	}
	if s == "" {
		return nil
	}
	newV, err := NewVersion(s)
	if err != nil {
		return fmt.Errorf("failed to unmarshal version: %w", err)
	}
	*v = *newV
	return nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (v *Version) UnmarshalYAML(f func(interface{}) error) error {
	return v.unmarshal(f)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *Version) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(strings.Trim(string(b), "\""))
	if s == "" || s == "null" {
		// go doesn't allow to set nil to a non-pointer struct field, so the result
		// is going to be an empty struct
		return nil
	}
	return v.unmarshal(func(i interface{}) error {
		*(i.(*string)) = s
		return nil
	})
}

func (v *Version) IsZero() bool {
	return v == nil || v.String() == ""
}

// Satisfies returns true if the version satisfies the supplied constraint
func (v *Version) Satisfies(constraint Constraints) bool {
	return constraint.Check(v)
}

// NewVersion returns a new Version created from the supplied string or an error if the string is not a valid version number
func NewVersion(v string) (*Version, error) {
	n, err := goversion.NewVersion(strings.TrimPrefix(v, "v"))
	if err != nil {
		return nil, err
	}

	return &Version{Version: *n}, nil
}

// MustParse is like NewVersion but panics if the version cannot be parsed.
// It simplifies safe initialization of global variables.
func MustParse(v string) *Version {
	version, err := NewVersion(v)
	if err != nil {
		panic("github.com/k0sproject/version: NewVersion: " + err.Error())
	}
	return version
}
