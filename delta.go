package version

import (
	"fmt"
)

// Delta represents the differences between two versions.
type Delta struct {
	a, b                *Version
	MajorUpgrade        bool
	MinorUpgrade        bool
	PatchUpgrade        bool
	K0sUpgrade          bool
	Equal               bool
	Downgrade           bool
	PrereleaseOnly      bool
	BuildMetadataChange bool
	Consecutive         bool
}

// NewDelta analyzes the differences between two versions and returns a Delta.
func NewDelta(a, b *Version) Delta {
	if a == nil || b == nil {
		panic("NewDelta called with a nil Version")
	}

	cmp := a.Compare(b)
	majorEqual, minorEqual, patchEqual := a.segmentEqual(b, 0), a.segmentEqual(b, 1), a.segmentEqual(b, 2)
	lessThan := cmp < 0

	d := Delta{
		a:                   a,
		b:                   b,
		MajorUpgrade:        lessThan && a.segments[0] < b.segments[0],
		MinorUpgrade:        lessThan && majorEqual && a.segments[1] < b.segments[1],
		PatchUpgrade:        lessThan && majorEqual && minorEqual && a.segments[2] < b.segments[2],
		Equal:               cmp == 0,
		Downgrade:           cmp > 0,
		K0sUpgrade:          majorEqual && minorEqual && patchEqual && a.pre == b.pre && a.isK0s && b.isK0s && a.k0s < b.k0s,
		PrereleaseOnly:      lessThan && a.Patch() == b.Patch() && (a.pre != "" || b.pre != ""),
		BuildMetadataChange: a.meta != b.meta,
	}

	switch {
	case d.PatchUpgrade:
		d.Consecutive = b.segments[2]-a.segments[2] == 1
	case d.MinorUpgrade:
		d.Consecutive = b.segments[1]-a.segments[1] == 1 && b.segments[2] == 0
	case d.MajorUpgrade:
		d.Consecutive = b.segments[0]-a.segments[0] == 1 && b.segments[1] == 0 && b.segments[2] == 0
	case d.K0sUpgrade:
		d.Consecutive = b.k0s-a.k0s == 1
	}

	return d
}

func (d Delta) conseq() string {
	if d.Consecutive {
		return "consecutive"
	}
	return "non-consecutive"
}

// String returns a human-readable representation of the Delta.
func (d Delta) String() string {
	if d.Downgrade {
		return fmt.Sprintf("%s is a downgrade from %s", d.b, d.a)
	}
	if d.MajorUpgrade {
		return fmt.Sprintf("a %s major upgrade from %s to %s", d.conseq(), d.a.Major(), d.b.Major())
	}
	if d.MinorUpgrade {
		return fmt.Sprintf("a %s minor upgrade from %s to %s", d.conseq(), d.a.Minor(), d.b.Minor())
	}
	if d.PrereleaseOnly {
		if d.b.pre == "" {
			return fmt.Sprintf("an upgrade from a %s pre-release to stable", d.a.Patch())
		}
		return fmt.Sprintf("an upgrade between pre-release versions of %s", d.a.Patch())
	}
	if d.PatchUpgrade {
		return fmt.Sprintf("a %s patch upgrade to %s", d.conseq(), d.b)
	}

	if d.K0sUpgrade {
		return fmt.Sprintf("a %s k0s upgrade to k0s build %d", d.conseq(), d.b.k0s)
	}

	if d.BuildMetadataChange {
		return fmt.Sprintf("build metadata changes from %q to %q", d.a.meta, d.b.meta)
	}

	return "no change"
}
