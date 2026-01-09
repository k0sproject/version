package version

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

type minorKey struct {
	major int
	minor int
}

func minorFromVersion(v *Version) minorKey {
	segments := v.Segments()
	return minorKey{major: segments[0], minor: segments[1]}
}

func compareMinor(a, b minorKey) int {
	switch {
	case a.major < b.major:
		return -1
	case a.major > b.major:
		return 1
	case a.minor < b.minor:
		return -1
	case a.minor > b.minor:
		return 1
	default:
		return 0
	}
}

// UpgradePath returns the recommended stable upgrade path from the receiver to target.
// It selects the latest stable patch for each minor along the way and appends the
// target when needed. Intermediate prereleases are skipped except when the target
// itself is a prerelease.
func (v *Version) UpgradePath(target *Version) (Collection, error) {
	if v == nil {
		return nil, errors.New("current version is nil")
	}
	if target == nil {
		return nil, errors.New("target version is nil")
	}
	if target.LessThan(v) {
		return nil, fmt.Errorf("target version %s is older than %s", target.String(), v.String())
	}

	all, err := All(context.Background())
	if err != nil {
		return nil, err
	}

	versionsByString := make(map[string]*Version, len(all))
	latestByMinor := make(map[minorKey]*Version)
	for _, candidate := range all {
		if candidate == nil {
			continue
		}
		key := candidate.String()
		versionsByString[key] = candidate
		if candidate.IsPrerelease() {
			continue
		}

		minor := minorFromVersion(candidate)
		if current, ok := latestByMinor[minor]; !ok || current.LessThan(candidate) {
			latestByMinor[minor] = candidate
		}
	}

	startMinor := minorFromVersion(v)
	targetMinor := minorFromVersion(target)

	keys := make([]minorKey, 0, len(latestByMinor))
	for key := range latestByMinor {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return compareMinor(keys[i], keys[j]) < 0
	})

	current := v
	path := Collection{}

	for _, key := range keys {
		if compareMinor(key, startMinor) < 0 {
			continue
		}
		if compareMinor(key, targetMinor) > 0 {
			break
		}

		candidate := latestByMinor[key]
		if candidate == nil {
			continue
		}

		if target.IsPrerelease() && compareMinor(key, targetMinor) == 0 && candidate.GreaterThan(target) {
			continue
		}

		if !target.IsPrerelease() && candidate.GreaterThan(target) {
			continue
		}

		if !candidate.GreaterThan(current) {
			continue
		}

		path = append(path, candidate)
		current = candidate
	}

	targetString := target.String()
	targetVersion := versionsByString[targetString]
	if targetVersion == nil {
		targetVersion = target
	}

	if target.IsPrerelease() {
		if current.LessThan(targetVersion) {
			path = append(path, targetVersion)
		} else if len(path) == 0 || path[len(path)-1].String() != targetString {
			path = append(path, targetVersion)
		}
	} else {
		if len(path) == 0 || path[len(path)-1].String() != targetString {
			if targetVersion.GreaterThan(current) {
				path = append(path, targetVersion)
			}
		}
	}

	deduped := Collection{}
	seen := make(map[string]struct{}, len(path))
	for _, candidate := range path {
		if candidate == nil {
			continue
		}
		if !candidate.GreaterThan(v) {
			continue
		}
		key := candidate.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, candidate)
	}

	return deduped, nil
}
