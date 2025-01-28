package version

import (
	"cmp"
	"fmt"
)

// MajorMinor represents the major and minor segments of a [Version].
//
// Useful for matching and comparing whole releases:
//
//	// RBAC is enabled by default since Kubernetes 1.6
//	var rbacEnabledByDefault = version.AtLeast(version.NewMajorMinor(1, 6))
//
//	k0sVersion := version.MustParse(...)
//	if k0sVersion.Is(rbacEnabledByDefault) {
//		... do something related to RBAC
//	}
type MajorMinor struct {
	major, minor uint
}

func NewMajorMinor(major, minor uint) MajorMinor {
	return MajorMinor{major, minor}
}

// Extracts the major and minor version segments of v.
// Non-existing segments in v are assumed to be zero.
func (v *Version) ToMajorMinor() MajorMinor {
	return NewMajorMinor(uint(v.segments[0]), uint(v.segments[1]))
}

// Returns the string representation of m.
// For example, the following will return "1.2":
//
//	NewMajorMinor(1, 2).String()
func (m MajorMinor) String() string {
	return fmt.Sprintf("%d.%d", m.major, m.minor)
}

// Implements [VersionMatcher]: v matches when its major and minor version
// segments are equal to m's major and minor segments.
func (m MajorMinor) MatchVersion(v *Version) bool {
	return m == v.ToMajorMinor()
}

// Compares m to other, first the major segment, then the minor segment.
func (m MajorMinor) Compare(other MajorMinor) int {
	if cmp := cmp.Compare(m.major, other.major); cmp != 0 {
		return cmp
	}
	return cmp.Compare(m.minor, other.minor)
}

// Implements [VersionComparer]: Compares m to v by only considering v's major
// and minor version segments. Non-existing segments in v are assumed to be zero.
func (m MajorMinor) CompareVersion(v *Version) int {
	return m.Compare(v.ToMajorMinor())
}
