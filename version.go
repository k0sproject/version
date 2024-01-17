// package version implements a k0s version type and functions to parse and compare versions
package version

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	BaseUrl     = "https://github.com/k0sproject/k0s/"
	k0s         = "k0s"
	maxSegments = 3
)

// this contains the fields that can be compared using go's equality operator
type comparableFields struct {
	// arrays (not slices) of basic types are comparable in go
	segments    [maxSegments]int
	numSegments int

	pre   string
	isK0s bool
	k0s   int
	meta  string
}

// Version is a k0s version
type Version struct {
	comparableFields
	s string
}

// NewVersion returns a new Version object from a string representation of a k0s version
func NewVersion(v string) (*Version, error) {
	if len(v) > 0 && v[0] == 'v' {
		v = v[1:]
	}
	if v == "" {
		return nil, errors.New("empty version")
	}
	for _, c := range v {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '+' && c != '-' && c != '.' {
			// version can only contain a-z, 0-9, +, -, .
			return nil, fmt.Errorf("can't contain character %c", c)
		}
	}
	idx := strings.IndexAny(v, "-+")
	var extra string
	if idx >= 0 {
		extra = v[idx:]
		v = v[:idx]
	}
	segments := strings.Split(v, ".")
	if len(segments) > maxSegments {
		return nil, fmt.Errorf("too many segments (%d > %d", len(segments), maxSegments)
	}

	version := &Version{comparableFields: comparableFields{numSegments: len(segments)}}
	for idx, s := range segments {
		segment, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parsing segment '%s': %w", s, err)
		}
		version.segments[idx] = int(segment)
	}

	if extra == "" {
		return version, nil
	}

	var minusIndex int
	plusIndex := strings.Index(extra, "+")
	if plusIndex == -1 {
		minusIndex = strings.Index(extra, "-")
	} else {
		minusIndex = strings.Index(extra[:plusIndex], "-")
	}

	if minusIndex != -1 {
		if plusIndex == -1 {
			// no meta
			version.pre = extra[minusIndex+1:]
		} else {
			version.pre = extra[minusIndex+1 : plusIndex]
		}
	}

	if plusIndex == -1 {
		return version, nil
	}

	meta := extra[plusIndex+1:]
	metaParts := strings.Split(meta, ".")
	if len(metaParts) == 1 {
		version.meta = meta
	} else {
		// parse the k0s.<version> part from metadata
		// and rebuild a new metadata string without it
		var newMeta strings.Builder
		for idx, part := range metaParts {
			if part == k0s && idx < len(metaParts)-1 {
				k0sV, err := strconv.ParseUint(metaParts[idx+1], 10, 32)
				if err == nil {
					version.isK0s = true
					version.k0s = int(k0sV)
				}
			} else if idx > 0 && metaParts[idx-1] != k0s {
				newMeta.WriteString(part)
				if idx < len(metaParts)-1 {
					newMeta.WriteString(".")
				}
			}
		}
		version.meta = newMeta.String()
	}

	return version, nil
}

// Segments returns the numerical segments of the k0s version (eg 1.2.3 from v1.2.3).
func (v *Version) Segments() []int {
	return v.segments[:v.numSegments]
}

// Prerelease returns the prerelease part of the k0s version (eg rc1 from v1.2.3-rc1).
func (v *Version) Prerelease() string {
	return v.pre
}

// IsK0s returns true if the version is a k0s version
func (v *Version) IsK0s() bool {
	return v.isK0s
}

// K0sVersion returns the k0s version (eg 4 from v1.2.3-k0s.4)
func (v *Version) K0sVersion() int {
	return v.k0s
}

// Metadata returns the metadata part of the k0s version (eg 123abc from v1.2.3+k0s.1.123abc)
func (v *Version) Metadata() string {
	return v.meta
}

// ComparableFields returns the comparable fields of the k0s version
func (v *Version) ComparableFields() comparableFields {
	return v.comparableFields
}

// Segments64 returns the numerical segments of the k0s version as int64 (eg 1.2.3 from v1.2.3).
func (v *Version) Segments64() []int64 {
	segments := make([]int64, v.numSegments)
	for i := 0; i < v.numSegments; i++ {
		segments[i] = int64(v.segments[i])
	}
	return segments
}

// IsPrerelease returns true if the k0s version is a prerelease version
func (v *Version) IsPrerelease() bool {
	return v.pre != ""
}

// String returns a v-prefixed string representation of the k0s version
func (v *Version) String() string {
	if v == nil {
		return ""
	}
	if v.s != "" {
		return v.s
	}
	if v.numSegments == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteRune('v')
	for i := 0; i < v.numSegments; i++ {
		sb.WriteString(strconv.Itoa(v.segments[i]))
		if i < v.numSegments-1 {
			sb.WriteRune('.')
		}
	}
	if v.pre != "" {
		sb.WriteRune('-')
		sb.WriteString(v.pre)
	}
	if v.isK0s || v.meta != "" {
		sb.WriteRune('+')
	}
	if v.isK0s {
		sb.WriteString(k0s)
		sb.WriteRune('.')
		sb.WriteString(strconv.Itoa(v.k0s))
		if v.meta != "" {
			sb.WriteRune('.')
		}
	}
	if v.meta != "" {
		sb.WriteString(v.meta)
	}

	v.s = sb.String()
	return v.s
}

// Equal returns true if the k0s version is equal to the supplied version
func (v *Version) Equal(b *Version) bool {
	if v == nil || b == nil {
		// nil versions are not equal
		return false
	}

	if v.s != "" && b.s != "" {
		// compare strings if both versions are already stringified
		return v.s == b.s
	}

	// compare comparable fields using go's equality operator
	return v.comparableFields == b.comparableFields
}

// Compare returns 0 if the k0s version is equal to the supplied version, 1 if it's greater and -1 if it's lower
func (v *Version) Compare(b *Version) int {
	if v.Equal(b) {
		return 0
	}
	for i := 0; i < maxSegments; i++ {
		if v.numSegments >= i+1 && b.numSegments >= i+1 {
			if v.segments[i] > b.segments[i] {
				return 1
			}
			if v.segments[i] < b.segments[i] {
				return -1
			}
		}
		if i >= v.numSegments && i < b.numSegments {
			// b has more segments, so it's greater
			return -1
		}
		if i >= b.numSegments && i < v.numSegments {
			// v has more segments, so it's greater
			return 1
		}
	}
	if v.pre == "" && b.pre != "" {
		return 1
	}
	if v.pre != "" && b.pre == "" {
		return -1
	}
	// segments are equal, so compare pre
	if v.pre < b.pre {
		return -1
	}
	if v.pre > b.pre {
		return 1
	}
	if v.isK0s && !b.isK0s {
		return 1
	}
	if !v.isK0s && b.isK0s {
		return -1
	}
	if v.k0s > b.k0s {
		return 1
	}
	if v.k0s < b.k0s {
		return -1
	}
	// meta should not affect precedence
	return 0
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

// GreaterThan returns true if the version is greater than the supplied version
func (v *Version) GreaterThan(b *Version) bool {
	return v.Compare(b) == 1
}

// LessThan returns true if the version is lower than the supplied version
func (v *Version) LessThan(b *Version) bool {
	return v.Compare(b) == -1
}

// GreaterThanOrEqual returns true if the version is greater than the supplied version or equal
func (v *Version) GreaterThanOrEqual(b *Version) bool {
	return v.Compare(b) >= 0
}

// LessThanOrEqual returns true if the version is lower than the supplied version or equal
func (v *Version) LessThanOrEqual(b *Version) bool {
	return v.Compare(b) <= 0
}

// MarshalText implements the encoding.TextMarshaler interface (used as fallback by encoding/json and yaml.v3).
func (v *Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface (used as fallback by encoding/json and yaml.v3).
func (v *Version) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*v = Version{}
		return nil
	}
	version, err := NewVersion(string(text))
	if err != nil {
		return err
	}
	*v = *version

	return nil
}

// MarshalYAML implements the yaml.v2 Marshaler interface.
func (v *Version) MarshalYAML() (interface{}, error) {
	if v == nil || v.numSegments == 0 {
		return nil, nil
	}
	return v.String(), nil
}

// UnmarshalYAML implements the yaml.v2 Unmarshaler interface.
func (v *Version) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var text string
	if err := unmarshal(&text); err != nil {
		return err
	}
	return v.UnmarshalText([]byte(text))
}

// IsZero returns true if the version is nil or empty
func (v *Version) IsZero() bool {
	return v == nil || v.numSegments == 0
}

// Satisfies returns true if the version satisfies the supplied constraint
func (v *Version) Satisfies(constraint Constraints) bool {
	return constraint.Check(v)
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
